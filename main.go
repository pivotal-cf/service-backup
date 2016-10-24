package main

import (
	"os"

	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf-experimental/service-backup/config"
	"github.com/tedsuo/ifrit"
	"gopkg.in/robfig/cron.v2"
)

func main() {
	logger := lager.NewLogger("ServiceBackup")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	configPath := os.Args[1]
	executor, cronSchedule, _ := config.Parse(configPath, logger)

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
