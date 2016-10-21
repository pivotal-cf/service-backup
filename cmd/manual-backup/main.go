package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf-experimental/service-backup/config"
)

var (
	logger lager.Logger
)

func main() {
	logger := lager.NewLogger("ServiceBackup")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	configPath := os.Args[1]
	executor, _, _ := config.Parse(configPath, logger)

	if executor == nil {
		return
	}

	if err := executor.RunOnce(); err != nil {
		log.Fatalf("error running backup: %s\n", err)
	}
}
