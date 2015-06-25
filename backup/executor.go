package backup

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pivotal-cf-experimental/service-backup/s3"
	"github.com/pivotal-golang/lager"
)

type Executor interface {
	RunOnce() error
}

type backup struct {
	s3Client         s3.S3Client
	sourceFolder     string
	destBucket       string
	destPath         string
	backupCreatorCmd string
	cleanupCmd       string
	logger           lager.Logger
}

func NewExecutor(
	s3Client s3.S3Client,
	sourceFolder,
	destBucket,
	destPath,
	backupCreatorCmd,
	cleanupCmd string,
	logger lager.Logger,
) Executor {
	return &backup{
		s3Client:         s3Client,
		sourceFolder:     sourceFolder,
		destBucket:       destBucket,
		destPath:         destPath,
		backupCreatorCmd: backupCreatorCmd,
		cleanupCmd:       cleanupCmd,
		logger:           logger,
	}
}

func (b *backup) RunOnce() error {
	err := b.createBucketIfNeeded()
	if err != nil {
		return err
	}

	err = b.performBackup()
	if err != nil {
		return err
	}

	err = b.uploadBackup()
	if err != nil {
		return err
	}

	// Do not return error if cleanup command failed.
	_ = b.performCleanup()
	return nil
}

func (b *backup) createBucketIfNeeded() error {
	b.logger.Info("Checking for bucket", lager.Data{"destBucket": b.destBucket})
	bucketExists, err := b.s3Client.BucketExists(b.destBucket)
	if err != nil {
		return err
	}

	if bucketExists {
		return nil
	}

	b.logger.Info("Checking for bucket - bucket does not exist - making it now")
	err = b.s3Client.CreateBucket(b.destBucket)
	if err != nil {
		return err
	}
	b.logger.Info("Checking for bucket - bucket created ok")
	return nil
}

func (b *backup) performBackup() error {
	b.logger.Info("Perform backup started")
	args := strings.Split(b.backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	b.logger.Debug("Perform backup debug info", lager.Data{"cmd": b.backupCreatorCmd, "out": string(out)})

	if err != nil {
		b.logger.Error("Perform backup completed with error", err)
		return err
	}

	b.logger.Info("Perform backup completed without error")
	return nil
}

func (b *backup) performCleanup() error {
	if b.cleanupCmd == "" {
		b.logger.Info("Cleanup command not provided")
		return nil
	}
	b.logger.Info("Cleanup started")

	args := strings.Split(b.cleanupCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	b.logger.Debug("Cleanup debug info", lager.Data{"cmd": b.cleanupCmd, "out": string(out)})

	if err != nil {
		b.logger.Error("Cleanup completed with error", err)
		return err
	}

	b.logger.Info("Cleanup completed without error")
	return nil
}

func (b *backup) uploadBackup() error {
	b.logger.Info("Upload backup started")

	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	destPathWithDate := b.destPath + "/" + datePath

	err := b.s3Client.Sync(
		b.sourceFolder,
		fmt.Sprintf("%s/%s", b.destBucket, destPathWithDate),
	)

	if err != nil {
		b.logger.Error("Upload backup completed with error", err)
		return err
	}

	b.logger.Info("Upload backup completed without error")
	return nil
}
