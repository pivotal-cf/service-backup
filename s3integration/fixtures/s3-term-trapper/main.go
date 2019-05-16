package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/shared"
)

func main() {
	if len(os.Args) < 6 {
		fmt.Fprintf(os.Stderr, "Usage: %s blah*5 <evidenceFile> s3://<startFile>", os.Args[0])
		os.Exit(1)
	}

	colonSplit := strings.Split(os.Args[6], "://")
	startFile := colonSplit[1]

	evidenceFile := os.Args[5]

	fmt.Println("startedFile", startFile)

	timeout := "10000"
	shared.InterruptibleSleep(evidenceFile, startFile, timeout, "0")
}
