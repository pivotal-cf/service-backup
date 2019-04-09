package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/shared"
)

func main() {
	if len(os.Args) < 8 {
		fmt.Fprintf(os.Stderr, "Usage: %s blah blah <startFile> blah blah blah <evidenceFile>", os.Args[0])
		os.Exit(1)
	}

	colonSplit := strings.Split(os.Args[7], ":")
	evidenceFile := colonSplit[0]
	startFile := colonSplit[1]

	fmt.Println("startedFile", startFile)

	timeout := "10000"
	shared.InterruptibleSleep(evidenceFile, startFile, timeout)
}
