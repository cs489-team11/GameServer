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
) {
	flag.Parse()
	if flag.NArg() < 8 {
		fmt.Println("Got less arguments than expected")
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
	parseArgs(
		&servAddr,
		&duration,
		&playerPoints,
		&bankPointsPerPlayer,
		&creditInterest,
		&depositInterest,
		&creditTime,
		&depositTime,
	)

	if creditInterest <= depositInterest {
		fmt.Printf(
			"Credit interest (%d) has to be larger than deposit interest (%d).\n",
			creditInterest,
			depositInterest,
		)
		os.Exit(1)
	}

	if creditTime >= duration || depositTime >= duration {
		fmt.Printf(
			"Credit (%d) and deposit (%d) times have to be less than duration of a game (%d).\n",
			creditTime,
			depositTime,
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
	)

	s := server.NewServer(gameConfig)
	if _, err := s.Listen(servAddr); err != nil {
		log.Fatalf("Server failed to listen: %v", err)
	}
	s.Launch()
}
