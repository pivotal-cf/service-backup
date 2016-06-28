package backup

import (
	"fmt"
	"strings"
)

type MultiBackuper []Backuper

func (m MultiBackuper) Upload(localPath string) error {
	var errors []error
	for _, b := range m {
		err := b.Upload(localPath)
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

func (m MultiBackuper) SetLogSession(sessionName, sessionIdentifier string) {
	for _, b := range m {
		b.SetLogSession(sessionName, sessionIdentifier)
	}
}

func (m MultiBackuper) CloseLogSession() {
	for _, b := range m {
		b.CloseLogSession()
	}
}
