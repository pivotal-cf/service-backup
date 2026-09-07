// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package s3_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager/v3"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/upload"
)

// requiresExplicitChecksum reports whether the given request is one of the
// three S3 operations that carry a checksum-algorithm header (PutObject,
// CreateMultipartUpload, UploadPart). CompleteMultipartUpload and bucket
// operations (HeadBucket, CreateBucket) never carry one and must not be
// rejected by the strict gateway below.
func requiresExplicitChecksum(r *http.Request) bool {
	pathParts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	hasKey := len(pathParts) == 2 && pathParts[1] != ""
	if !hasKey {
		return false
	}

	switch r.Method {
	case http.MethodPut:
		// PutObject or UploadPart (with ?partNumber=&uploadId=).
		return true
	case http.MethodPost:
		// CreateMultipartUpload (?uploads); CompleteMultipartUpload
		// (?uploadId= only, no "uploads" key) must not match.
		_, isCreateMultipart := r.URL.Query()["uploads"]
		return isCreateMultipart
	default:
		return false
	}
}

// newStrictChecksumGateway starts an httptest.Server that reproduces the
// behavior reported in TNZ-127170 against a Huawei OceanStor Pacific
// endpoint: any PutObject/CreateMultipartUpload/UploadPart request that does
// not carry an explicit SHA256 checksum algorithm is rejected with the same
// error the customer saw. Everything else (including a correctly-checksummed
// retry) is reverse-proxied through to a real S3-compatible backend so
// bucket/object semantics stay real rather than mocked.
func newStrictChecksumGateway(backendURL string) *httptest.Server {
	target, err := url.Parse(backendURL)
	Expect(err).NotTo(HaveOccurred())

	proxy := httputil.NewSingleHostReverseProxy(target)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requiresExplicitChecksum(r) {
			algo := r.Header.Get("X-Amz-Checksum-Algorithm")
			if algo == "" {
				algo = r.Header.Get("X-Amz-Sdk-Checksum-Algorithm")
			}
			if !strings.EqualFold(algo, "SHA256") {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`+
					`<Error><Code>InvalidRequest</Code>`+
					`<Message>The checksum algorithm must SHA256</Message>`+
					`<RequestId>strict-gateway-test</RequestId></Error>`)
				return
			}
		}
		proxy.ServeHTTP(w, r)
	}))
}

var _ = Describe("strict checksum gateway (TNZ-127170 repro against a real S3-compatible backend)", func() {
	var (
		backendEndpoint, accessKey, secretKey string
		gateway                               *httptest.Server
		logger                                lager.Logger
		bucket                                 string
	)

	BeforeEach(func() {
		backendEndpoint = os.Getenv("MINIO_ENDPOINT")
		accessKey = os.Getenv("MINIO_ACCESS_KEY")
		secretKey = os.Getenv("MINIO_SECRET_KEY")
		if backendEndpoint == "" || accessKey == "" || secretKey == "" {
			Skip("set MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY to run this against a real S3-compatible backend")
		}

		logger = lager.NewLogger("strict-gateway-test")
		bucket = "service-backup-checksum-repro"
		gateway = newStrictChecksumGateway(backendEndpoint)

		client, err := s3.CreateS3Client(logger, accessKey, secretKey, gateway.URL, "us-east-1", true)
		Expect(err).NotTo(HaveOccurred())

		setupClient := s3.New("setup", "", gateway.URL, "us-east-1", accessKey, secretKey, "", true, upload.RemotePathFunc(bucket, ""))
		Expect(setupClient.CreateBucketIfNeeded(client, bucket+"/setup", logger)).To(Succeed())
	})

	AfterEach(func() {
		gateway.Close()
	})

	writeTempFile := func(sizeBytes int) string {
		tmpFile, err := os.CreateTemp("", "strict-gateway-*.txt")
		Expect(err).NotTo(HaveOccurred())
		content := strings.Repeat("a", sizeBytes)
		_, err = tmpFile.WriteString(content)
		Expect(err).NotTo(HaveOccurred())
		Expect(tmpFile.Close()).To(Succeed())
		return tmpFile.Name()
	}

	Context("single-part upload (small file, PutObject)", func() {
		var tmpFilePath string

		BeforeEach(func() {
			tmpFilePath = writeTempFile(1024)
		})

		AfterEach(func() {
			os.Remove(tmpFilePath)
		})

		It("succeeds automatically via the SHA256 retry, with no configuration required", func() {
			client, err := s3.CreateS3Client(logger, accessKey, secretKey, gateway.URL, "us-east-1", true)
			Expect(err).NotTo(HaveOccurred())

			s3Client := s3.New("name", "", gateway.URL, "us-east-1", accessKey, secretKey, "", true, upload.RemotePathFunc("", ""))
			err = s3Client.UploadFile(logger, client, tmpFilePath, bucket+"/small.txt")

			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("multipart upload (large file, CreateMultipartUpload + UploadPart)", func() {
		var tmpFilePath string

		BeforeEach(func() {
			// Larger than the SDK's default multipart threshold (5MiB) so this
			// actually exercises CreateMultipartUpload/UploadPart, not PutObject.
			tmpFilePath = writeTempFile(6 * 1024 * 1024)
		})

		AfterEach(func() {
			os.Remove(tmpFilePath)
		})

		It("succeeds automatically via the SHA256 retry, with no configuration required", func() {
			client, err := s3.CreateS3Client(logger, accessKey, secretKey, gateway.URL, "us-east-1", true)
			Expect(err).NotTo(HaveOccurred())

			s3Client := s3.New("name", "", gateway.URL, "us-east-1", accessKey, secretKey, "", true, upload.RemotePathFunc("", ""))
			err = s3Client.UploadFile(logger, client, tmpFilePath, bucket+"/large.txt")

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
