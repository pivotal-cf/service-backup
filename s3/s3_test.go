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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"code.cloudfoundry.org/lager/v3"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/upload"
)


var _ = Describe("S3", func() {
	Describe("default arguments", func() {
		var (
			lsCmd                                                                       *exec.Cmd
			awsCmdPath, endpointURL, region, accessKey, secretKey, systemTrustStorePath string
		)

		BeforeEach(func() {
			awsCmdPath = "path/to/aws-cli"
			endpointURL = "http://example.com"
			region = "aws-region"
			accessKey = "access-key"
			secretKey = "secret-key"
			systemTrustStorePath = "path/to/system/trust/store"
		})

		JustBeforeEach(func() {
			s3CLIClient := s3.New("destination-name", awsCmdPath, endpointURL, region, accessKey, secretKey, systemTrustStorePath, false, upload.RemotePathFunc("base-path", ""))
			lsCmd = s3CLIClient.S3Cmd("ls", "bucket-name")
		})

		It("builds an S3 command with default arguments", func() {
			Expect(lsCmd.Args).To(Equal([]string{
				awsCmdPath,
				"--endpoint-url",
				endpointURL,
				"--region",
				region,
				"s3",
				"ls",
				"bucket-name",
			}))
		})

		Context("when endpoint URL is empty", func() {
			BeforeEach(func() {
				endpointURL = ""
			})

			It("builds an S3 command without specifying endpoint url", func() {
				Expect(lsCmd.Args).To(Equal([]string{
					awsCmdPath,
					"--region",
					region,
					"s3",
					"ls",
					"bucket-name",
				}))
			})
		})

		Context("when region is empty", func() {
			BeforeEach(func() {
				region = ""
			})

			It("builds an S3 command without specifying region", func() {
				Expect(lsCmd.Args).To(Equal([]string{
					awsCmdPath,
					"--endpoint-url",
					endpointURL,
					"s3",
					"ls",
					"bucket-name",
				}))
			})
		})

		It("adds the AWS credentials to the S3 command's env", func() {
			Expect(lsCmd.Env).To(ContainElement(fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", accessKey)))
			Expect(lsCmd.Env).To(ContainElement(fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", secretKey)))
		})
	})

	Describe("CreateS3Client", func() {
		var (
			logger          lager.Logger
			receivedHeaders http.Header
			requestPaths    []string
			server          *httptest.Server
		)

		BeforeEach(func() {
			logger = lager.NewLogger("s3-test")
			receivedHeaders = nil
			requestPaths = nil

			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedHeaders = r.Header.Clone()
				requestPaths = append(requestPaths, r.URL.Path)
				// Respond with 200 for HeadBucket; 200 for everything else
				w.WriteHeader(http.StatusOK)
			}))
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when use_path_style is true", func() {
			It("uses path-style addressing so the bucket name is in the URL path", func() {
				client, err := s3.CreateS3Client(logger, "access", "secret", server.URL, "us-east-1", true)
				Expect(err).NotTo(HaveOccurred())

				s3Client := s3.New("name", "", server.URL, "us-east-1", "access", "secret", "", true, upload.RemotePathFunc("test-bucket", ""))
				_ = s3Client.CreateBucketIfNeeded(client, "test-bucket/prefix", logger)

				// With path-style, the bucket name appears as the first path segment
				Expect(requestPaths).To(ContainElement(HavePrefix("/test-bucket")))
			})
		})

		Context("when a custom endpoint is configured", func() {
			It("does not send extra payload checksum headers (x-amz-checksum-*) that non-AWS S3 endpoints reject", func() {
				client, err := s3.CreateS3Client(logger, "access", "secret", server.URL, "us-east-1", true)
				Expect(err).NotTo(HaveOccurred())

				tmpFile, err := os.CreateTemp("", "s3-test-*.txt")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(tmpFile.Name())
				_, err = tmpFile.WriteString("test content")
				Expect(err).NotTo(HaveOccurred())
				tmpFile.Close()

				s3Client := s3.New("name", "", server.URL, "us-east-1", "access", "secret", "", true, upload.RemotePathFunc("", ""))
				_ = s3Client.UploadFile(logger, client, tmpFile.Name(), "test-bucket/test-key")

				// RequestChecksumCalculationWhenRequired suppresses the extra payload checksum headers
				// (x-amz-checksum-crc32, x-amz-checksum-sha256, etc.) that AWS SDK v2 adds by default.
				// Non-AWS S3-compatible endpoints do not support these headers and reject uploads with
				// errors like XAmzContentSHA256Mismatch.
				Expect(receivedHeaders.Get("x-amz-checksum-crc32")).To(BeEmpty())
				Expect(receivedHeaders.Get("x-amz-checksum-sha256")).To(BeEmpty())

				// The signing hash (x-amz-content-sha256) must still be present and contain the
				// actual payload hash (not UNSIGNED-PAYLOAD or STREAMING-AWS4-HMAC-SHA256-PAYLOAD).
				sha256 := receivedHeaders.Get("x-amz-content-sha256")
				Expect(sha256).NotTo(BeEmpty())
				Expect(sha256).NotTo(Equal("UNSIGNED-PAYLOAD"))
				Expect(sha256).NotTo(ContainSubstring("STREAMING"))
			})
		})

		Context("when no custom endpoint is configured (standard AWS)", func() {
			It("creates a client without error", func() {
				client, err := s3.CreateS3Client(logger, "access", "secret", "", "us-east-1", false)
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
			})
		})

		Context("when the endpoint rejects the first attempt for lacking a checksum algorithm", func() {
			It("retries automatically with an explicit SHA256 checksum and succeeds, with no configuration required", func() {
				var attempts []http.Header

				server.Close()
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPut || !strings.Contains(r.URL.Path, "test-key") {
						w.WriteHeader(http.StatusOK)
						return
					}

					attempts = append(attempts, r.Header.Clone())
					if len(attempts) == 1 {
						w.Header().Set("Content-Type", "application/xml")
						w.WriteHeader(http.StatusBadRequest)
						fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`+
							`<Error><Code>InvalidRequest</Code>`+
							`<Message>The checksum algorithm must SHA256</Message>`+
							`<RequestId>test-request-id</RequestId></Error>`)
						return
					}
					w.WriteHeader(http.StatusOK)
				}))

				client, err := s3.CreateS3Client(logger, "access", "secret", server.URL, "us-east-1", true)
				Expect(err).NotTo(HaveOccurred())

				tmpFile, err := os.CreateTemp("", "s3-test-*.txt")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(tmpFile.Name())
				_, err = tmpFile.WriteString("test content")
				Expect(err).NotTo(HaveOccurred())
				tmpFile.Close()

				s3Client := s3.New("name", "", server.URL, "us-east-1", "access", "secret", "", true, upload.RemotePathFunc("", ""))
				err = s3Client.UploadFile(logger, client, tmpFile.Name(), "test-bucket/test-key")

				Expect(err).NotTo(HaveOccurred())
				Expect(attempts).To(HaveLen(2), "expected exactly one retry after the first rejection")
				Expect(attempts[0].Get("x-amz-checksum-sha256")).To(BeEmpty(), "first attempt should not carry an explicit checksum")
				Expect(attempts[1].Get("x-amz-checksum-sha256")).NotTo(BeEmpty(), "retry should carry an explicit SHA256 checksum")
			})
		})

		Context("when the endpoint rejects every attempt, including the SHA256 retry", func() {
			It("returns the underlying error rather than retrying indefinitely", func() {
				var attemptCount int

				server.Close()
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPut || !strings.Contains(r.URL.Path, "test-key") {
						w.WriteHeader(http.StatusOK)
						return
					}

					attemptCount++
					w.Header().Set("Content-Type", "application/xml")
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`+
						`<Error><Code>InvalidRequest</Code>`+
						`<Message>The checksum algorithm must SHA256</Message>`+
						`<RequestId>test-request-id</RequestId></Error>`)
				}))

				client, err := s3.CreateS3Client(logger, "access", "secret", server.URL, "us-east-1", true)
				Expect(err).NotTo(HaveOccurred())

				tmpFile, err := os.CreateTemp("", "s3-test-*.txt")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(tmpFile.Name())
				tmpFile.Close()

				s3Client := s3.New("name", "", server.URL, "us-east-1", "access", "secret", "", true, upload.RemotePathFunc("", ""))
				err = s3Client.UploadFile(logger, client, tmpFile.Name(), "test-bucket/test-key")

				Expect(err).To(MatchError(ContainSubstring("InvalidRequest: The checksum algorithm must SHA256")))
				Expect(attemptCount).To(Equal(2), "expected the initial attempt plus exactly one retry, not an infinite loop")
			})
		})
	})

	Describe("UploadDir", func() {
		var (
			logger    lager.Logger
			server    *httptest.Server
			uploadDir string
			callCount int
		)

		BeforeEach(func() {
			logger = lager.NewLogger("s3-test")
			callCount = 0

			var err error
			uploadDir, err = os.MkdirTemp("", "upload-dir-*")
			Expect(err).NotTo(HaveOccurred())

			Expect(os.WriteFile(filepath.Join(uploadDir, "file1.txt"), []byte("content1"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(uploadDir, "file2.txt"), []byte("content2"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(uploadDir, "file3.txt"), []byte("content3"), 0644)).To(Succeed())
		})

		AfterEach(func() {
			os.RemoveAll(uploadDir)
			server.Close()
		})

		Context("when one file upload fails but others succeed", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPut {
						callCount++
					}
					// Consistently fail requests for file1.txt regardless of retries,
					// using 400 (not retried by the SDK) to keep the test deterministic.
					if strings.Contains(r.URL.Path, "file1.txt") {
						http.Error(w, "InvalidDigest", http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusOK)
				}))
			})

			It("continues uploading remaining files and returns a combined error", func() {
				client, err := s3.CreateS3Client(logger, "access", "secret", server.URL, "us-east-1", true)
				Expect(err).NotTo(HaveOccurred())

				s3Client := s3.New("name", "", server.URL, "us-east-1", "access", "secret", "", true, upload.RemotePathFunc("", ""))
				uploadErr := s3Client.UploadDir(client, logger, uploadDir, "test-bucket/prefix")

				// Error is returned
				Expect(uploadErr).To(HaveOccurred())
				Expect(uploadErr.Error()).To(ContainSubstring("1 file(s) failed to upload"))

				// All 3 files were attempted (not aborted after first failure)
				Expect(callCount).To(Equal(3))
			})
		})

		Context("when all files upload successfully", func() {
			BeforeEach(func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			})

			It("returns no error", func() {
				client, err := s3.CreateS3Client(logger, "access", "secret", server.URL, "us-east-1", true)
				Expect(err).NotTo(HaveOccurred())

				s3Client := s3.New("name", "", server.URL, "us-east-1", "access", "secret", "", true, upload.RemotePathFunc("", ""))
				Expect(s3Client.UploadDir(client, logger, uploadDir, "test-bucket/prefix")).To(Succeed())
			})
		})
	})
})
