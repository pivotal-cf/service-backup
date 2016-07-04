package backup

import (
	"fmt"
	"strings"

	"github.com/pivotal-golang/lager"
)

type MultiBackuper []Backuper

func (m MultiBackuper) Upload(localPath string, logger lager.Logger) error {
	var errors []error
	for _, b := range m {
		err := b.Upload(localPath, logger)
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
