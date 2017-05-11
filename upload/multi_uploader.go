package upload

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/lager"
)

type multiUploader struct {
	uploaders []Uploader
}

func (m *multiUploader) Upload(localPath string, logger lager.Logger) error {
	var errors []error
	for _, u := range m.uploaders {
		sessionLogger := logger
		if u.Name() != "" {
			sessionLogger = logger.WithData(lager.Data{"destination_name": u.Name()})
		}
		err := u.Upload(localPath, sessionLogger)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return formattedError(errors)
}

func (m *multiUploader) Name() string {
	names := make([]string, len(m.uploaders))
	for i, u := range m.uploaders {
		names[i] = u.Name()
	}

	return fmt.Sprintf("multi-uploader: %s", strings.Join(names, ", "))
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
