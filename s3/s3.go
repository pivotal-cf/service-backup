// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package s3

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/aws/aws-sdk-go-v2/config"
	aws_s3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pivotal-cf/service-backup/process"
)

type S3CliClient struct {
	name         string
	awsCmdPath   string
	accessKey    string
	secretKey    string
	endpointURL  string
	region       string
	caCertPath   string
	remotePathFn func() string
	ProcessMgr   process.ProcessManager
}

func New(name, awsCmdPath, endpointURL, region, accessKey, secretKey, caCertPath string, remotePathFn func() string) *S3ClientConfig {
	return &S3CliClient{
		name:         name,
		awsCmdPath:   awsCmdPath,
		endpointURL:  endpointURL,
		region:       region,
		accessKey:    accessKey,
		secretKey:    secretKey,
		caCertPath:   caCertPath,
		remotePathFn: remotePathFn,
	}
}

func (c *S3CliClient) S3Cmd(args ...string) *aws_s3.Client {
	//var cmdArgs []string

	//var credentials = aws.Credentials{
	//	AccessKeyID:     c.accessKey,
	//	SecretAccessKey: c.secretKey,
	//}
	fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.accessKey)
	fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.secretKey)
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(c.region),
		config.WithCustomCABundle(strings.NewReader(c.caCertPath)),
	)

	if err != nil {
		log.Fatal(err)
	}

	//var envConfig = config.EnvConfig{
	//	Credentials:    credentials,
	//	Region:         c.region,
	//	CustomCABundle: c.caCertPath,
	//}

	//if c.endpointURL != "" {
	//	cmdArgs = append(cmdArgs, "--endpoint-url", c.endpointURL)
	//}

	//if c.region != "" {
	//	cmdArgs = append(cmdArgs, "--region", c.region)
	//}

	//cmdArgs = append(cmdArgs, "--ca-bundle", c.caCertPath)
	//cmdArgs = append(cmdArgs, "s3")
	//cmdArgs = append(cmdArgs, args...)

	//cmd := exec.Command(c.awsCmdPath, cmdArgs...)
	//cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.accessKey))
	//cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.secretKey))

	var client = aws_s3.NewFromConfig(cfg)
	return client
}

func (c *S3CliClient) CreateRemotePathIfNeeded(remotePath string, sessionLogger lager.Logger) error {
	sessionLogger.Info("Checking for remote path", lager.Data{"remotePath": remotePath})
	remotePathExists, err := c.remotePathExists(remotePath, sessionLogger)
	if err != nil {
		return err
	}

	if remotePathExists {
		return nil
	}

	sessionLogger.Info("Checking for remote path - remote path does not exist - making it now")
	err = c.createRemotePath(remotePath)
	if err != nil {
		if strings.Contains(err.Error(), "AccessDenied") {
			sessionLogger.Error("Configured S3 user unable to create buckets", err)
		}

		return err
	}
	sessionLogger.Info("Checking for remote path - remote path created ok")
	return nil
}

func (c *S3CliClient) remotePathExists(remotePath string, sessionLogger lager.Logger) (bool, error) {
	bucketName := strings.Split(remotePath, "/")[0]

	cmd := c.S3Cmd("ls", bucketName)
	cmd.GetBucketLocation()
	if out, err := c.ProcessMgr.Start(cmd); err != nil {
		if bytes.Contains(out, []byte("NoSuchBucket")) {
			return false, nil
		}

		wrappedErr := fmt.Errorf("unknown s3 error occurred: '%s' with output: '%s'", err, string(out))
		sessionLogger.Error("error checking if bucket exists", wrappedErr)
		return false, wrappedErr
	}

	return true, nil
}

func (c *S3CliClient) createRemotePath(remotePath string) error {
	bucketName := strings.Split(remotePath, "/")[0]
	cmd := c.S3Cmd("mb", fmt.Sprintf("s3://%s", bucketName))
	return c.RunCommand(cmd, "create bucket")
}

func (c *S3CliClient) Upload(localPath string, sessionLogger lager.Logger, processManager process.ProcessManager) error {
	defer sessionLogger.Info("s3 completed")

	c.ProcessMgr = processManager

	remotePath := c.remotePathFn()

	sessionLogger.Info(fmt.Sprintf("about to upload %s to S3 remote path %s", localPath, remotePath))
	cmd := c.S3Cmd("sync", localPath, fmt.Sprintf("s3://%s", remotePath))
	out, err := c.ProcessMgr.Start(cmd)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "SIGTERM") {
		return nil
	}
	if !bytes.Contains(out, []byte("NoSuchBucket")) {
		return fmt.Errorf("error in sync: %s, output: %s", err, string(out))
	}

	err = c.CreateRemotePathIfNeeded(remotePath, sessionLogger)
	if err != nil {
		return err
	}

	cmd = c.S3Cmd("sync", localPath, fmt.Sprintf("s3://%s", remotePath))
	return c.RunCommand(cmd, "sync")
}

func (c *S3CliClient) RunCommand(cmd *exec.Cmd, stepName string) error {
	if out, err := c.ProcessMgr.Start(cmd); err != nil {
		return fmt.Errorf("error in %s: %s, output: %s", stepName, err, string(out))
	}
	return nil
}

func (c *S3CliClient) Name() string {
	return c.name
}
