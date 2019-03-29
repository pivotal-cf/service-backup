package main

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/shared"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: %s <don't care> <evidenceFile> <don't care> <startedFile> [...]", os.Args[0])
		os.Exit(1)
	}

	evidenceFile, startFile := os.Args[2], os.Args[4]
	shared.InterruptibleSleep(evidenceFile, startFile, "20")
}
