package main

import (
	"log"
	"os"

	"github.com/pivotal-cf-experimental/service-backup/config"
	"github.com/pivotal-golang/lager"
)

var (
	logger lager.Logger
)

func main() {
	executor, _, _ := config.Parse(os.Args)

	if executor == nil {
		return
	}

	if err := executor.RunOnce(); err != nil {
		log.Fatalf("error running backup: %s\n", err)
	}
}
