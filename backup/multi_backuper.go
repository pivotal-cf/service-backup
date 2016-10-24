package backup

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/lager"
)

type Uploader struct {
	backupers []Backuper
}

func NewUploader(backupers []Backuper) Uploader {
	return Uploader{backupers: backupers}
}

func (m Uploader) Upload(localPath string, logger lager.Logger) error {
	var errors []error
	for _, b := range m.backupers {
		sessionLogger := logger
		if b.Name() != "" {
			sessionLogger = logger.WithData(lager.Data{"destination_name": b.Name()})
		}
		err := b.Upload(localPath, sessionLogger)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return formattedError(errors)
}

func formattedError(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	errorMessages := []string{}
	for _, e := range errors {
		errorMessages = append(errorMessages, e.Error())
	}
	return fmt.Errorf(strings.Join(errorMessages, "; "))
}
