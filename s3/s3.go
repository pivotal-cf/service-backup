// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/lager"
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

func New(name, awsCmdPath, endpointURL, region, accessKey, secretKey, caCertPath string, remotePathFn func() string) *S3CliClient {
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

func (c *S3CliClient) S3Cmd(args ...string) *exec.Cmd {
	var cmdArgs []string

	if c.endpointURL != "" {
		cmdArgs = append(cmdArgs, "--endpoint-url", c.endpointURL)
	}

	if c.region != "" {
		cmdArgs = append(cmdArgs, "--region", c.region)
	}

	cmdArgs = append(cmdArgs, "s3")
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(c.awsCmdPath, cmdArgs...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.accessKey))
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.secretKey))
	return cmd
}

func (c *S3CliClient) CreateBucketIfNeeded(client *s3.Client, remotePath string, sessionLogger lager.Logger) error {
	sessionLogger.Info("Checking for remote path", lager.Data{"remotePath": remotePath})
	remotePathExists, err := c.bucketExists(client, remotePath, sessionLogger)
	if err != nil {
		return err
	}

	if remotePathExists {
		return nil
	}

	sessionLogger.Info("Checking for remote path - remote path does not exist - making it now")
	err = c.createBucket(client, remotePath)
	if err != nil {
		if strings.Contains(err.Error(), "AccessDenied") {
			sessionLogger.Error("Configured S3 user unable to create buckets", err)
		}

		return err
	}
	sessionLogger.Info("Checking for remote path - remote path created ok")
	return nil
}

func (c *S3CliClient) bucketExists(client *s3.Client, fullRemoteFilePath string, sessionLogger lager.Logger) (bool, error) {
	remoteFilePathElements := strings.Split(fullRemoteFilePath, "/")
	bucketName := remoteFilePathElements[0]

	input := &s3.HeadBucketInput{
		Bucket: &bucketName,
	}

	_, err := client.HeadBucket(context.TODO(), input)
	if err != nil {
		var apiError smithy.APIError

		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				return false, nil
			default:
				return false, fmt.Errorf("bucketExists: api error, %v", err)
			}
		} else {
			return false, fmt.Errorf("bucketExists: error listing objects, %v", err)
		}
	}

	return true, nil
}

func (c *S3CliClient) createBucket(client *s3.Client, remotePath string) error {
	bucketName := strings.Split(remotePath, "/")[0]
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(c.region),
		},
	}
	_, err := client.CreateBucket(context.TODO(), input)

	return err
}

func CreateS3Client(sessionLogger lager.Logger, accessKey, secretKey, endpointURL, region string) (*s3.Client, error) {
	if len(region) == 0 {
		sessionLogger.Info("CreateS3Client: ===warning=== region is empty. therefore using default region us-west-2")
		region = "us-west-2"
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           endpointURL,
			SigningRegion: region,
			Source:        aws.EndpointSourceCustom,
		}, nil
	})

	var cfg aws.Config
	var err error
	if len(endpointURL) == 0 {
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	} else {
		sessionLogger.Info("using a custom endpoint is deprecated with the aws sdk")
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
			config.WithEndpointResolverWithOptions(customResolver))
	}

	if err != nil {
		return nil, fmt.Errorf("UploadDir: failed to load SDK configuration, %v", err)
	}

	client := s3.NewFromConfig(cfg)

	return client, nil
}

func (c *S3CliClient) Upload(localPath string, sessionLogger lager.Logger, processManager process.ProcessManager) error {
	defer sessionLogger.Info("s3 completed")

	c.ProcessMgr = processManager

	remotePath := c.remotePathFn()

	sessionLogger.Info(fmt.Sprintf("about to upload %s to S3 remote path %s", localPath, remotePath))

	client, err := CreateS3Client(sessionLogger, c.accessKey, c.secretKey, c.endpointURL, c.region)
	if err != nil {
		return fmt.Errorf("upload: couldn't create client: %v", err)
	}

	err = c.CreateBucketIfNeeded(client, remotePath, sessionLogger)
	if err != nil {
		return err
	}

	return c.UploadDir(client, sessionLogger, localPath, remotePath)
}

func (c *S3CliClient) Name() string {
	return c.name
}

func (c *S3CliClient) UploadFile(logger lager.Logger, client *s3.Client, localFilePath, fullRemoteFilePath string) error {
	remoteFilePathElements := strings.Split(fullRemoteFilePath, "/")
	bucketName := remoteFilePathElements[0]
	remotePath := strings.Join(remoteFilePathElements[1:], "/")

	logger.Info(fmt.Sprintf("S3 putting local file: %s into bucket %s with remote file: %s ", localFilePath, bucketName, remotePath))

	readFile, err := os.ReadFile(localFilePath)
	if err != nil {
		return fmt.Errorf("UploadFile: failed to read local file path: %v", err)
	}
	largeBuffer := bytes.NewReader(readFile)
	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &remotePath,
		Body:   largeBuffer,
	})
	if err != nil {
		return fmt.Errorf("UploadFile: failed to put object: %v", err)
	}

	return nil
}

func (c *S3CliClient) UploadDir(client *s3.Client, logger lager.Logger, localDir string, remotePath string) error {
	err := filepath.Walk(localDir, func(filePath string, d os.FileInfo, err error) error {
		if d.IsDir() {
			return nil
		}

		relativeFilePath := strings.Replace(filePath, localDir, "", -1)
		remoteFilePath := filepath.Join(remotePath, relativeFilePath)

		return c.UploadFile(logger, client, filePath, remoteFilePath)
	})
	if err != nil {
		return fmt.Errorf("UploadDir: failed to walk dir, %v", err)
	}

	return nil
}
