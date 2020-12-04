package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/cs489-team11/server"
)

func parseArgs(
	servAddr *string,
	duration *int32,
	playerPoints *int32,
	bankPointsPerPlayer *int32,
	creditInterest *int32,
	depositInterest *int32,
	creditTime *int32,
	depositTime *int32,
	theftTime *int32,
	theftPercentage *int32,
	lotteryTime *int32,
	lotteryMaxWin *int32,
) {
	flag.Parse()
	receivedArgs := flag.NArg()
	requiredArgs := 12
	if receivedArgs < requiredArgs {
		fmt.Printf(
			"Got less arguments than expected. Have: %d. Want: %d.\n",
			receivedArgs,
			requiredArgs,
		)
		os.Exit(1)
	}

	*servAddr = flag.Arg(0)

	arg1, err := strconv.Atoi(flag.Arg(1))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(1))
		os.Exit(2)
	}
	*duration = int32(arg1)

	arg2, err := strconv.Atoi(flag.Arg(2))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(2))
		os.Exit(2)
	}
	*playerPoints = int32(arg2)

	arg3, err := strconv.Atoi(flag.Arg(3))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(3))
		os.Exit(2)
	}
	*bankPointsPerPlayer = int32(arg3)

	arg4, err := strconv.Atoi(flag.Arg(4))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(4))
		os.Exit(2)
	}
	*creditInterest = int32(arg4)

	arg5, err := strconv.Atoi(flag.Arg(5))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(5))
		os.Exit(2)
	}
	*depositInterest = int32(arg5)

	arg6, err := strconv.Atoi(flag.Arg(6))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(6))
		os.Exit(2)
	}
	*creditTime = int32(arg6)

	arg7, err := strconv.Atoi(flag.Arg(7))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(7))
		os.Exit(2)
	}
	*depositTime = int32(arg7)

	arg8, err := strconv.Atoi(flag.Arg(8))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(8))
		os.Exit(2)
	}
	*theftTime = int32(arg8)

	arg9, err := strconv.Atoi(flag.Arg(9))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(9))
		os.Exit(2)
	}
	*theftPercentage = int32(arg9)

	arg10, err := strconv.Atoi(flag.Arg(10))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(10))
		os.Exit(2)
	}
	*lotteryTime = int32(arg10)

	arg11, err := strconv.Atoi(flag.Arg(11))
	if err != nil {
		fmt.Printf("%s is not an integer\n", flag.Arg(11))
		os.Exit(2)
	}
	*lotteryMaxWin = int32(arg11)
}

func main() {
	var servAddr string // for localhost, it needs to be "0.0.0.0:9090"
	var duration int32
	var playerPoints int32
	var bankPointsPerPlayer int32
	var creditInterest int32
	var depositInterest int32
	var creditTime int32
	var depositTime int32
	var theftTime int32
	var theftPercentage int32
	var lotteryTime int32
	var lotteryMaxWin int32
	parseArgs(
		&servAddr,
		&duration,
		&playerPoints,
		&bankPointsPerPlayer,
		&creditInterest,
		&depositInterest,
		&creditTime,
		&depositTime,
		&theftTime,
		&theftPercentage,
		&lotteryTime,
		&lotteryMaxWin,
	)

	if creditInterest <= depositInterest {
		fmt.Printf(
			"Credit interest (%d) has to be larger than deposit interest (%d).\n",
			creditInterest,
			depositInterest,
		)
		os.Exit(1)
	}

	if creditInterest >= 100 || depositInterest >= 100 || theftPercentage >= 100 {
		fmt.Printf(
			"Credit (%d), deposit (%d), theft (%d) percentages have to be less than 100 percent.\n",
			creditInterest,
			depositInterest,
			theftPercentage,
		)
		os.Exit(1)
	}

	if creditTime >= duration || depositTime >= duration || theftTime >= duration || lotteryTime >= duration {
		fmt.Printf(
			"Credit (%d)sec, deposit (%d)sec, theft (%d)sec, lottery (%d)sec times have to be less than duration of a game (%d).\n",
			creditTime,
			depositTime,
			theftTime,
			lotteryTime,
			duration,
		)
		os.Exit(1)
	}

	gameConfig := server.NewGameConfig(
		duration,
		playerPoints,
		bankPointsPerPlayer,
		creditInterest,
		depositInterest,
		creditTime,
		depositTime,
		theftTime,
		theftPercentage,
		lotteryTime,
		lotteryMaxWin,
	)

	s := server.NewServer(gameConfig)
	if _, err := s.Listen(servAddr); err != nil {
		log.Fatalf("Server failed to listen: %v", err)
	}
	s.Launch()
}
