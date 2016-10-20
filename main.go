package main

import (
	"os"

	"github.com/pivotal-cf-experimental/service-backup/config"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"gopkg.in/robfig/cron.v2"
)

var (
	logger lager.Logger
)

func main() {
	executor, cronSchedule, _, logger := config.Parse(os.Args)

	if executor == nil {
		return
	}

	scheduler := cron.New()

	_, err := scheduler.AddFunc(cronSchedule, func() {
		executor.RunOnce()
	})

	if err != nil {
		logger.Error("Error scheduling job", err)
		os.Exit(2)
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
		logger.Error("Error running", err)
		os.Exit(2)
	}
}
