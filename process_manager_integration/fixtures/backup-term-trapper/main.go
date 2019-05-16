package main

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/shared"
)

func main() {
	if len(os.Args) < 4 || len(os.Args) > 5 {
		fmt.Fprintf(os.Stderr, "Usage: %s <startFile> <evidenceFile> <sleepyTime> [sleepyTimeAfterSigterm]", os.Args[0])
		os.Exit(1)
	}

	evidenceFile, startFile, timeout := os.Args[1], os.Args[2], os.Args[3]

	sleepyTimeAfterSigterm := "0"
	if len(os.Args) == 5 {
		sleepyTimeAfterSigterm = os.Args[4]
	}
	shared.InterruptibleSleep(evidenceFile, startFile, timeout, sleepyTimeAfterSigterm)
}
