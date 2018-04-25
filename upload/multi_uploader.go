// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package upload

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/process"
)

type multiUploader struct {
	uploaders []Uploader
}

func (m *multiUploader) Upload(localPath string, logger lager.Logger, processManager process.ProcessManager) error {
	var errors []error
	for _, u := range m.uploaders {
		sessionLogger := logger
		if u.Name() != "" {
			sessionLogger = logger.WithData(lager.Data{"destination_name": u.Name()})
		}
		err := u.Upload(localPath, sessionLogger, processManager)
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
