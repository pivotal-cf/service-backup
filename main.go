package main

import (
	"os"

	"github.com/pivotal-cf-experimental/service-backup/parseargs"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"gopkg.in/robfig/cron.v2"
)

var (
	logger lager.Logger
)

func main() {
	executor, cronSchedule, logger := parseargs.Parse(os.Args)

	if executor == nil {
		return
	}

	scheduler := cron.New()

	_, err := scheduler.AddFunc(*cronSchedule, func() {
		executor.RunOnce()
	})

	if err != nil {
		logger.Fatal("Error scheduling job", err)
	}

	schedulerRunner := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		scheduler.Start()
		close(ready)

		<-signals
		scheduler.Stop()
		return nil
	})

	process := ifrit.Invoke(schedulerRunner)
	logger.Info("Service-backup Started")

	err = <-process.Wait()
	if err != nil {
		logger.Fatal("Error running", err)
	}
}
