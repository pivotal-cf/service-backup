package main

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/shared"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <startFile> <evidenceFile> <sleepyTime>", os.Args[0])
		os.Exit(1)
	}

	evidenceFile, startFile, timeout := os.Args[1], os.Args[2], os.Args[3]
	shared.InterruptibleSleep(evidenceFile, startFile, timeout)
}
