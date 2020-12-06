package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cs489-team11/server/pb"
	"github.com/google/uuid"
)

type questionID string
type userID string
type username string

type questionInfo struct {
	bidPoints     int32
	correctAnswer int32 // index of correct answer from 1 to 4
}

// this struct does not have RWMutex as its member
// this is done to avoid deadlocks between goroutines
type player struct {
	userID            userID
	username          username
	points            int32
	stream            pb.Game_StreamServer
	gameStartNotified bool
	lastLotteryTime   time.Time
	questions         map[questionID]*questionInfo
}

func newQuestionInfo(
	bidPoints int32,
	correctAnswer int32,
) *questionInfo {
	return &questionInfo{
		bidPoints,
		correctAnswer,
	}
}

func newPlayer(username username, points int32) *player {
	userID := userID(uuid.New().String())
	return &player{
		userID:            userID,
		username:          username,
		points:            points,
		stream:            nil,
		gameStartNotified: false,
		lastLotteryTime:   time.Now(),
		questions:         make(map[questionID]*questionInfo),
	}
}

// when game calls this function on player, make sure to grab
// WRITE lock on game
func (p *player) setStream(stream pb.Game_StreamServer) {
	p.stream = stream
	log.Printf("Stream for user %v has been set.\n", p.userID)
}

// when game calls this function on player, make sure to grab
// WRITE lock on game
func (p *player) updateLastLotteryTime() {
	p.lastLotteryTime = time.Now()
}

// when game calls this function on player, make sure to grab
// READ lock on game
// "lotteryTime" is the time in seconds from game config,
// which has to pass before player can play lottery again
func (p *player) canPlayLottery(lotteryTime int32) bool {
	return time.Since(p.lastLotteryTime) >= (time.Duration(lotteryTime) * time.Second / time.Nanosecond)
}

func (p *player) generateQuestion(bidPoints int32) (questionID, string, []string, error) {
	if bidPoints > p.points {
		return "", "", nil, fmt.Errorf(
			"bid points (%d) has to be less than or equal to player's points (%d)",
			bidPoints,
			p.points,
		)
	}

	resp, err := http.Get("https://opentdb.com/api.php?amount=1&difficulty=easy&type=multiple&encode=base64")
	if err != nil {
		return "", "", nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	/*
		Sample API response body:
			map[response_code:0 results:[map[category:Entertainment: Musicals & Theatres correct_answer:Et tu, Brute?  difficulty:easy
			incorrect_answers:[Iacta alea est! Vidi, vini, vici. Aegri somnia vana.] question:In Shakespeare&#039;s play Julius Caesa
			r, Caesar&#039;s last words were... type:multiple]]]
	*/

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", "", nil, fmt.Errorf("response decoding failure: %v", err)
	}
	//log.Printf("Question response data:\n%v\n", results_[0])

	// parse API response body
	results := data["results"].([]interface{})[0].(map[string]interface{})
	question := decodeB64(results["question"].(string))
	correctAnswer := decodeB64(results["correct_answer"].(string))
	incorrectAnswers := make([]string, 3)
	for i := 0; i < 3; i++ {
		incorrectAnswers[i] = decodeB64(results["incorrect_answers"].([]interface{})[i].(string))
	}

	correctAnswerIndex := seededRand.Intn(4) // 0,1,2, or 3
	allAnswers := insertToSlice(incorrectAnswers, correctAnswerIndex, correctAnswer)

	questionID := questionID(uuid.New().String())
	qInfo := newQuestionInfo(bidPoints, int32(correctAnswerIndex+1))
	p.questions[questionID] = qInfo

	return questionID, question, allAnswers, nil
}

func (p *player) answerQuestion(
	questionID questionID, userAnswer int32,
) (
	bool, int32, int32, error,
) {
	qInfo, ok := p.questions[questionID]
	if !ok {
		errMsg := fmt.Sprintf("there is no question %v for player %v", questionID, p.userID)
		return false, 0, 0, fmt.Errorf(errMsg)
	}

	return qInfo.correctAnswer == userAnswer, qInfo.correctAnswer, qInfo.bidPoints, nil
}

// when game calls this function on player, make sure to grab
// READ lock on game
func (p *player) toPBPlayer() *pb.Player {
	return &pb.Player{
		UserId:   string(p.userID),
		Username: string(p.username),
		Points:   p.points,
	}
}
