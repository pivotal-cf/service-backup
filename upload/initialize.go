// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package upload

import (
	"fmt"

	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf/service-backup/azure"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/gcs"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/scp"
)

type Uploader interface {
	Upload(localPath string, sessionLogger lager.Logger, processManager process.ProcessManager) error
	Name() string
}

//go:generate counterfeiter -o fakes/uploader_factory.go . UploaderFactory
type UploaderFactory interface {
	S3(destination config.Destination, caCertPath string) *s3.S3CliClient
	SCP(destination config.Destination) *scp.SCPClient
	Azure(destination config.Destination) *azure.AzureClient
	GCS(destination config.Destination) *gcs.StorageClient
}

func Initialize(conf *config.BackupConfig, logger lager.Logger, options ...Option) (*multiUploader, error) {
	opts := &opts{
		factory:       &uploaderFactory{backupConfig: conf},
		caCertLocator: CACertPath,
	}

	for _, opt := range options {
		opt(opts)
	}

	uploaders := make([]Uploader, len(conf.Destinations))

	for i, dest := range conf.Destinations {
		switch dest.Type {
		case "s3":
			caCert, err := opts.caCertLocator()
			if err != nil {
				return nil, err
			}
			uploaders[i] = opts.factory.S3(dest, caCert)
		case "scp":
			uploaders[i] = opts.factory.SCP(dest)
		case "azure":
			uploaders[i] = opts.factory.Azure(dest)
		case "gcs":
			uploaders[i] = opts.factory.GCS(dest)
		default:
			err := fmt.Errorf("unknown destination type: %s", dest.Type)
			logger.Error("error parsing destinations", err)
			return nil, err
		}
	}

	return &multiUploader{uploaders}, nil
}
