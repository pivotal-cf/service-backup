package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)

	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <startFile> <evidenceFile> <sleepyTime>", os.Args[0])
		os.Exit(1)
	}

	evidenceFile, startFile, timeout := os.Args[1], os.Args[2], os.Args[3]
	createFile(startFile)

	sleepyTime := convertToInteger(timeout)

	select {
	case <-time.After(time.Millisecond * time.Duration(sleepyTime)):
		log.Println("Time is up - exiting")
		os.Exit(1)
	case <-signalChan:
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
