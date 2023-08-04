// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package s3testclient

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/upload"
)

type S3TestClient struct {
	*s3.S3CliClient
}

func New(endpointURL, accessKeyID, secretAccessKey, basePath, region string) *S3TestClient {
	caCertPath, err := upload.CACertPath()
	Expect(err).NotTo(HaveOccurred())

	s3CLIClient := s3.New("s3_test_client", "aws", endpointURL, region, accessKeyID, secretAccessKey, caCertPath, upload.RemotePathFunc(basePath, ""))
	s3CLIClient.ProcessMgr = process.NewManager()
	return &S3TestClient{S3CliClient: s3CLIClient}
}

func (c *S3TestClient) ListRemotePath(bucketName, region string) ([]string, error) {
	cmdArgs := []string{}
	if region != "" {
		cmdArgs = append(cmdArgs, "--region", region)
	}
	cmdArgs = append(cmdArgs, "ls", "--recursive", fmt.Sprintf("s3://%s/", bucketName))
	cmd := c.S3Cmd(cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return []string{}, fmt.Errorf("command failed: %s\nwith error:%s", string(out), err)
	}

	remoteKeys := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		cols := strings.Fields(line)
		if len(cols) < 4 {
			continue
		}
		remoteKeys = append(remoteKeys, cols[3])
	}
	return remoteKeys, nil
}

func (c *S3TestClient) RemotePathExistsInBucket(bucketName, remotePath string) bool {
	keys, err := c.ListRemotePath(bucketName, "us-west-2")
	Expect(err).ToNot(HaveOccurred())

	for _, key := range keys {
		if strings.Contains(key, remotePath) {
			return true
		}
	}
	return false
}

func (c *S3TestClient) DownloadRemoteDirectory(bucketName, remotePath, localPath string) error {
	err := os.MkdirAll(localPath, 0777)
	if err != nil {
		return err
	}

	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "sync", fmt.Sprintf("s3://%s/%s", bucketName, remotePath), localPath)
	return c.RunCommand(cmd, "download remote")
}

func (c *S3TestClient) DeleteRemotePath(bucketName, remotePath, region string) error {
	cmd := c.S3Cmd()
	if region != "" {
		cmd.Args = append(cmd.Args, "--region", region)
	}
	cmd.Args = append(cmd.Args, "rm", "--recursive", fmt.Sprintf("s3://%s/%s", bucketName, remotePath))
	return c.RunCommand(cmd, "delete remote path")
}

func (c *S3TestClient) DeleteBucket(bucketName, region string) {
	err := c.DeleteRemotePath(bucketName, "", region)
	if err != nil && strings.Contains(err.Error(), "NoSuchBucket") {
		return
	}
	Expect(err).ToNot(HaveOccurred())

	rbArgs := []string{}
	if region != "" {
		rbArgs = append(rbArgs, "--region", region)
	}
	rbArgs = append(rbArgs, "rb", "--force", fmt.Sprintf("s3://%s", bucketName))

	cmd := c.S3Cmd(rbArgs...)

	err = c.RunCommand(cmd, "delete bucket")
	if err != nil {
		// Try again, because s3 is flaky
		time.Sleep(10 * time.Second)
		cmd = c.S3Cmd(rbArgs...)
		err = c.RunCommand(cmd, "retry delete bucket")
		Expect(err).ToNot(HaveOccurred())
	}
}

func (c *S3TestClient) RunCommand(cmd *exec.Cmd, stepName string) error {
	if out, err := c.ProcessMgr.Start(cmd); err != nil {
		return fmt.Errorf("error in %s: %s, output: %s", stepName, err, string(out))
	}
	return nil
}
