package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/shared"
)

func main() {
	if len(os.Args) != 10 {
		fmt.Fprintf(os.Stderr, "Usage: %s <something>*8 <evidenceFile@something:startedFile>", os.Args[0])
		os.Exit(1)
	}

	combinedArg := os.Args[9]
	atSplit := strings.Split(combinedArg, "@")
	colonSplit := strings.Split(combinedArg, ":")
	evidenceFile := atSplit[0]
	startFile := colonSplit[1]
	timeout := "10000"
	shared.InterruptibleSleep(evidenceFile, startFile, timeout, "0")
}
