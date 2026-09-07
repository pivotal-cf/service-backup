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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/lager/v3"
	"github.com/pivotal-cf/service-backup/process"
)

type S3CliClient struct {
	name         string
	awsCmdPath   string
	accessKey    string
	secretKey    string
	endpointURL  string
	region       string
	usePathStyle bool
	caCertPath   string
	remotePathFn func() string
	ProcessMgr   process.ProcessManager
}

func New(name, awsCmdPath, endpointURL, region, accessKey, secretKey, caCertPath string, usePathStyle bool, remotePathFn func() string) *S3CliClient {
	return &S3CliClient{
		name:         name,
		awsCmdPath:   awsCmdPath,
		endpointURL:  endpointURL,
		region:       region,
		accessKey:    accessKey,
		secretKey:    secretKey,
		usePathStyle: usePathStyle,
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
	}

	// us-east-1 is AWS's special default: the API rejects any LocationConstraint
	// for it. All other regions require a LocationConstraint. The operator is
	// responsible for providing the correct region via the tile UI (the UI
	// already states that region is required for any non-us-east-1 endpoint).
	if c.region != "" && c.region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(c.region),
		}
	}

	_, err := client.CreateBucket(context.TODO(), input)
	return err
}

func CreateS3Client(sessionLogger lager.Logger, accessKey, secretKey, endpointURL, region string, usePathStyle bool) (*s3.Client, error) {
	if len(region) == 0 {
		region = "us-east-1"
		sessionLogger.Info("CreateS3Client: region is empty, defaulting to us-east-1")
	}

	// Only send the x-amz-content-sha256 checksum header when the S3 server
	// explicitly requires it. The default WhenSupported sends it unconditionally,
	// which causes HTTP 400 (XAmzContentSHA256Mismatch) on non-AWS S3-compatible
	// endpoints that do not support the header.
	checksumOpt := config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired)

	var cfg aws.Config
	var err error
	if len(endpointURL) == 0 {
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
			checksumOpt)
	} else {
		sessionLogger.Info("using a custom endpoint is deprecated with the aws sdk")
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           endpointURL,
				SigningRegion: region,
				Source:        aws.EndpointSourceCustom,
			}, nil
		})
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
			config.WithEndpointResolverWithOptions(customResolver),
			checksumOpt)
	}

	if err != nil {
		return nil, fmt.Errorf("CreateS3Client: failed to load SDK configuration, %v", err)
	}

	var clientOpts []func(*s3.Options)
	if usePathStyle {
		clientOpts = append(clientOpts, func(o *s3.Options) { o.UsePathStyle = true })
	}
	client := s3.NewFromConfig(cfg, clientOpts...)

	return client, nil
}

func (c *S3CliClient) Upload(localPath string, sessionLogger lager.Logger, processManager process.ProcessManager) error {
	defer sessionLogger.Info("s3 completed")

	c.ProcessMgr = processManager

	remotePath := c.remotePathFn()

	sessionLogger.Info(fmt.Sprintf("about to upload %s to S3 remote path %s", localPath, remotePath))

	client, err := CreateS3Client(sessionLogger, c.accessKey, c.secretKey, c.endpointURL, c.region, c.usePathStyle)
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
	fileReader := bytes.NewReader(readFile)

	// manager.Uploader has its own RequestChecksumCalculation setting that
	// does NOT inherit from the *s3.Client's config (set in CreateS3Client) -
	// it defaults to WhenSupported independently. Without this, PutObject/
	// CreateMultipartUpload/UploadPart calls made through the uploader still
	// carry an unrequested CRC32 checksum header regardless of the client
	// config, which strict S3-compatible endpoints (e.g. Huawei OceanStor
	// Pacific) reject.
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})

	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &remotePath,
		Body:   fileReader,
	})

	if err != nil && isChecksumAlgorithmRejection(err) {
		// Some endpoints reject a request that carries no checksum at all,
		// rather than merely rejecting the wrong algorithm. Retry once with
		// an explicit SHA256 checksum, which every S3-compatible
		// implementation we're aware of accepts.
		logger.Info("upload rejected due to a missing/unsupported checksum algorithm; retrying with an explicit SHA256 checksum", lager.Data{
			"bucket": bucketName,
			"key":    remotePath,
		})

		if _, seekErr := fileReader.Seek(0, io.SeekStart); seekErr != nil {
			return fmt.Errorf("UploadFile: failed to rewind file for checksum retry: %v", seekErr)
		}

		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:            &bucketName,
			Key:               &remotePath,
			Body:              fileReader,
			ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
		})
	}

	if err != nil {
		return fmt.Errorf("UploadFile: failed to put object: %v", err)
	}

	return nil
}

// isChecksumAlgorithmRejection reports whether err is the class of error some
// strict S3-compatible endpoints (e.g. Huawei OceanStor Pacific) return when
// they require an explicit checksum algorithm on the request rather than
// accepting the absence of one, e.g.:
//
//	InvalidRequest: The checksum algorithm must SHA256
func isChecksumAlgorithmRejection(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.ErrorCode() == "InvalidRequest" &&
		strings.Contains(strings.ToLower(apiErr.ErrorMessage()), "checksum")
}

func (c *S3CliClient) UploadDir(client *s3.Client, logger lager.Logger, localDir string, remotePath string) error {
	var uploadErrors []error

	walkErr := filepath.Walk(localDir, func(filePath string, d os.FileInfo, err error) error {
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Errorf("walk error at %s: %v", filePath, err))
			return nil
		}
		if d.IsDir() {
			return nil
		}

		relativeFilePath := strings.Replace(filePath, localDir, "", -1)
		remoteFilePath := filepath.Join(remotePath, relativeFilePath)

		if ferr := c.UploadFile(logger, client, filePath, remoteFilePath); ferr != nil {
			uploadErrors = append(uploadErrors, ferr)
		}
		return nil
	})

	if walkErr != nil {
		return fmt.Errorf("UploadDir: failed to walk dir, %v", walkErr)
	}
	if len(uploadErrors) > 0 {
		return fmt.Errorf("UploadDir: %d file(s) failed to upload: %v", len(uploadErrors), uploadErrors)
	}

	return nil
}
