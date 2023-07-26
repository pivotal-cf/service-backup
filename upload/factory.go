// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package upload

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/service-backup/azure"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/gcs"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/scp"
)

type uploaderFactory struct {
	backupConfig *config.BackupConfig
}

func (b *uploaderFactory) S3(destination config.Destination, caCertPath string) *s3.S3CliClient {
	basePath := fmt.Sprintf(
		"%s/%s",
		toString(destination.Config["bucket_name"]),
		toString(destination.Config["bucket_path"]),
	)

	return s3.New(
		destination.Name,
		b.backupConfig.AwsCliPath,
		toString(destination.Config["endpoint_url"]),
		toString(destination.Config["region"]),
		toString(destination.Config["access_key_id"]),
		toString(destination.Config["secret_access_key"]),
		caCertPath,
		RemotePathFunc(basePath, b.backupConfig.DeploymentName),
	)
}

func (b *uploaderFactory) SCP(destination config.Destination) *scp.SCPClient {
	return scp.New(
		destination.Name,
		toString(destination.Config["server"]),
		toInt(destination.Config["port"]),
		toString(destination.Config["user"]),
		toString(destination.Config["key"]),
		toString(destination.Config["fingerprint"]),
		RemotePathFunc(toString(destination.Config["destination"]), b.backupConfig.DeploymentName),
	)
}

func (b *uploaderFactory) Azure(destination config.Destination) *azure.AzureClient {
	return azure.New(
		destination.Name,
		toString(destination.Config["storage_access_key"]),
		toString(destination.Config["storage_account"]),
		toString(destination.Config["container"]),
		toString(destination.Config["endpoint"]),
		RemotePathFunc(toString(destination.Config["path"]), b.backupConfig.DeploymentName),
	)
}

func (b *uploaderFactory) GCS(destination config.Destination) *gcs.StorageClient {
	return gcs.New(
		destination.Name,
		os.Getenv("GCP_SERVICE_ACCOUNT_FILE"),
		toString(destination.Config["project_id"]),
		toString(destination.Config["bucket_name"]),
		RemotePathFunc("", b.backupConfig.DeploymentName),
	)
}

func toString(raw interface{}) string {
	var value string
	if v, ok := raw.(string); ok {
		value = v
	}
	return value
}

func toInt(raw interface{}) int {
	var value int
	if v, ok := raw.(int); ok {
		value = v
	}
	return value
}
