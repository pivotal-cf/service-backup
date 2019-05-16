package shared

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func InterruptibleSleep(evidenceFile, startedFile, timeout, sleepAfterSigterm string) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)

	createFile(startedFile)

	sleepyTime := convertToInteger(timeout)
	sleepySigTermTime := convertToInteger(sleepAfterSigterm)

	select {
	case <-time.After(time.Millisecond * time.Duration(sleepyTime)):
		log.Println("Time is up - exiting")
		os.Exit(1)
	case <-signalChan:
		time.Sleep(time.Millisecond * time.Duration(sleepySigTermTime))
		log.Println("Caught a SIGTERM - exiting")
		createFile(evidenceFile)
		os.Exit(129)
	}
}

func convertToInteger(timeout string) int {
	sleepyTime, err := strconv.Atoi(timeout)
	if err != nil {
		log.Printf("couldn't convert arg %q to an int\n", timeout)
		log.Fatal(err)
	}

	return sleepyTime
}

func createFile(name string) {
	_, err := os.Create(name)
	if err != nil {
		log.Printf("couldn't create file %q", name)
		log.Fatal(err)
	}
}
