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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/lager/v3"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
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

// RegionFromEndpoint extracts the AWS region from a standard S3 endpoint URL.
//
// Handles:
//
//	https://s3.REGION.amazonaws.com   → REGION          (path-style / newer format)
//	https://s3-REGION.amazonaws.com   → REGION          (legacy format)
//	https://s3.amazonaws.com          → "us-east-1"     (global endpoint)
//	anything else (custom S3)         → ""              (non-AWS, no region)
func RegionFromEndpoint(endpointURL string) string {
	if !strings.Contains(endpointURL, ".amazonaws.com") {
		return ""
	}
	host := strings.TrimPrefix(strings.TrimPrefix(endpointURL, "https://"), "http://")
	// Strip trailing slash or path
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	switch {
	case host == "s3.amazonaws.com":
		return "us-east-1"
	case strings.HasPrefix(host, "s3."):
		// s3.REGION.amazonaws.com
		r := strings.TrimSuffix(strings.TrimPrefix(host, "s3."), ".amazonaws.com")
		if r != "" {
			return r
		}
	case strings.HasPrefix(host, "s3-"):
		// s3-REGION.amazonaws.com  (legacy)
		r := strings.TrimSuffix(strings.TrimPrefix(host, "s3-"), ".amazonaws.com")
		if r != "" {
			return r
		}
	}
	return ""
}

func (c *S3CliClient) createBucket(client *s3.Client, remotePath string) error {
	bucketName := strings.Split(remotePath, "/")[0]
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// For non-AWS custom endpoints LocationConstraint is not meaningful — the
	// server either ignores it or rejects it.
	if c.endpointURL != "" && !strings.Contains(c.endpointURL, ".amazonaws.com") {
		_, err := client.CreateBucket(context.TODO(), input)
		return err
	}

	// Determine effective region in priority order:
	//   1. explicitly configured region
	//   2. region inferred from the endpoint URL
	//   3. us-east-1 (AWS global default when no endpoint is specified)
	effectiveRegion := c.region
	if effectiveRegion == "" {
		effectiveRegion = RegionFromEndpoint(c.endpointURL)
	}
	if effectiveRegion == "" {
		effectiveRegion = "us-east-1"
	}

	// us-east-1 is AWS's special default: the API rejects any LocationConstraint
	// for it. All other regions require a LocationConstraint that matches the
	// endpoint's region.
	if effectiveRegion != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(effectiveRegion),
		}
	}

	_, err := client.CreateBucket(context.TODO(), input)
	return err
}

func CreateS3Client(sessionLogger lager.Logger, accessKey, secretKey, endpointURL, region string) (*s3.Client, error) {
	if len(region) == 0 {
		inferred := RegionFromEndpoint(endpointURL)
		if inferred != "" {
			region = inferred
			sessionLogger.Info("CreateS3Client: region not set, inferred from endpoint URL: " + region)
		} else {
			region = "us-east-1"
			sessionLogger.Info("CreateS3Client: ===warning=== region is empty and could not be inferred from endpoint. using default region us-east-1")
		}
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
	if len(endpointURL) > 0 && RegionFromEndpoint(endpointURL) == "" {
		// Non-AWS custom endpoints (e.g. MinIO, Ceph, NetApp, DBS) require
		// path-style addressing (http://host/bucket/key).  Standard AWS
		// endpoints use virtual-hosted-style (http://bucket.s3.region.amazonaws.com)
		// which is the AWS-recommended approach and avoids path-style deprecation
		// warnings for new buckets.
		clientOpts = append(clientOpts, func(o *s3.Options) { o.UsePathStyle = true })
	}
	client := s3.NewFromConfig(cfg, clientOpts...)

	return client, nil
}

// getBucketRegion returns the AWS region that owns bucketName.
//
// GetBucketLocation is the correct API for cross-region discovery: it is
// designed to return the bucket's LocationConstraint from *any* regional
// endpoint, responding with 200 OK + XML body rather than a 301 redirect.
//
// A plain us-east-1 client (no custom endpoint URL) is always created here so
// the SDK routes through s3.us-east-1.amazonaws.com — the standard global
// endpoint. Callers must NOT pass a client built from a legacy custom endpoint
// (e.g. https://s3-us-west-2.amazonaws.com) because those endpoints return
// NoSuchBucket for buckets in a different region instead of the location data.
//
// Returns the bucket region string, or "" if it could not be determined.
func getBucketRegion(accessKey, secretKey, bucketName string, logger lager.Logger) string {
	discoveryClient, err := CreateS3Client(logger, accessKey, secretKey, "", "us-east-1")
	if err != nil {
		logger.Info("getBucketRegion: could not create discovery client: " + err.Error())
		return ""
	}

	out, err := discoveryClient.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		logger.Info("getBucketRegion: GetBucketLocation failed: " + err.Error())
		return ""
	}

	// AWS returns an empty LocationConstraint for us-east-1 buckets.
	location := string(out.LocationConstraint)
	if location == "" {
		location = "us-east-1"
	}
	logger.Info("getBucketRegion: discovered region " + location)
	return location
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

	// When no region is configured and the endpoint is either absent or a
	// standard AWS S3 URL (not a custom S3-compatible system), discover the
	// bucket's actual region via HeadBucket (X-Amz-Bucket-Region header) before proceeding.
	//
	// This is necessary because:
	//  - With an empty endpoint, the client defaults to us-east-1.  Buckets in
	//    other regions cause HeadBucket/PutObject to receive HTTP 301/307.
	//  - With a real AWS regional endpoint that doesn't match the bucket's
	//    region (e.g. s3-us-west-2 for a us-east-1 bucket), path-style
	//    HeadBucket returns HTTP 404, and a subsequent CreateBucket call fails
	//    with BucketAlreadyExists.
	//
	// Once the actual region is known we recreate the client WITHOUT the custom
	// endpoint URL so that the SDK uses its standard virtual-hosted-style
	// regional endpoint (e.g. bucket.s3.us-east-1.amazonaws.com), which routes
	// correctly to the bucket regardless of how it was originally addressed.
	//
	// For custom S3-compatible endpoints (non-AWS) the region must be supplied
	// explicitly; no auto-discovery is attempted.
	if c.region == "" && (c.endpointURL == "" || RegionFromEndpoint(c.endpointURL) != "") {
		bucketName := strings.Split(remotePath, "/")[0]
		if actualRegion := getBucketRegion(c.accessKey, c.secretKey, bucketName, sessionLogger); actualRegion != "" {
			// Always drop the custom endpoint URL here: the discovered region
			// is authoritative, and AWS standard endpoint resolution handles
			// routing to the correct regional endpoint correctly.
			client, err = CreateS3Client(sessionLogger, c.accessKey, c.secretKey, "", actualRegion)
			if err != nil {
				return fmt.Errorf("upload: couldn't create client for discovered region %s: %v", actualRegion, err)
			}
		}
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
	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &remotePath,
		Body:   fileReader,
	})
	if err != nil {
		return fmt.Errorf("UploadFile: failed to put object: %v", err)
	}

	return nil
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
