// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package release_tests_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/pivotal-cf/service-backup/s3testclient"
	"github.com/pivotal-cf/service-backup/testhelpers"

	gcs "cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/api/option"
)

var _ = Describe("release tests", func() {
	const (
		bucketName       = "service-backup-test"
		testPath         = "release-tests"
		testSourceFolder = "/tmp/to_upload/"
	)

	var (
		boshEnv            string
		boshHost           string
		boshPrivateKeyFile string
		toBackup           string
		boshSSHUser        string
		boshManifest       string
	)

	type deployment struct {
		Name string
	}

	BeforeEach(func() {
		boshEnv = envMustHave("BOSH_ENV")
		boshHost = envMustHave("BOSH_HOST")
		boshPrivateKeyFile = envMustHave("BOSH_PRIVATE_KEY_FILE")
		boshSSHUser = envMustHave("BOSH_SSH_USER")
		toBackup = "to_backup.txt"
	})

	boshCmdWithGateway := func(stdout io.Writer, command string, args ...string) {
		dep := deployment{}

		manifestBytes, err := ioutil.ReadFile(boshManifest)
		Expect(err).NotTo(HaveOccurred())
		Expect(yaml.Unmarshal(manifestBytes, &dep)).To(Succeed())

		commonArgs := []string{
			"-e", boshEnv,
			"-d", dep.Name,
			command,
			"--gw-host", boshHost,
			"--gw-user", boshSSHUser,
			"--gw-private-key", boshPrivateKeyFile,
		}
		allArgs := append(commonArgs, args...)
		GinkgoWriter.Write([]byte(fmt.Sprintf("running BOSH SSH command %s\n", allArgs)))
		cmd := exec.Command("bosh", allArgs...)
		cmd.Stdout = stdout
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed())
	}

	boshSSH := func(command string) string {
		buf := new(bytes.Buffer)
		writer := io.MultiWriter(GinkgoWriter, buf)
		boshCmdWithGateway(writer, "ssh", append([]string{"service-backup/0", "-c"}, command)...)
		return buf.String()
	}

	boshSCP := func(source, destination string) {
		boshCmdWithGateway(GinkgoWriter, "scp", source, fmt.Sprintf("%s:%s", "service-backup/0", destination))
	}

	JustBeforeEach(func() {
		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		pathToFile := filepath.Join(cwd, "test_assets", toBackup)
		boshSSH(fmt.Sprintf("sudo chmod 775 %s", testSourceFolder))
		boshSCP(pathToFile, testSourceFolder)
	})

	Context("backing up to S3", func() {
		var (
			client *s3testclient.S3TestClient
		)

		BeforeEach(func() {
			boshManifest = envMustHave("S3_BOSH_MANIFEST")

			awsAccessKeyID := envMustHave("AWS_ACCESS_KEY_ID")
			awsSecretKey := envMustHave("AWS_SECRET_ACCESS_KEY")
			client = s3testclient.New("https://s3.amazonaws.com", awsAccessKeyID, awsSecretKey, testPath)
		})

		AfterEach(func() {
			boshSSH(fmt.Sprintf("rm -f %s%s", testSourceFolder, toBackup))
			Expect(client.DeleteRemotePath(bucketName, testPath, "")).To(Succeed())
		})

		Context("automatic backup", func() {
			It("Uploads files in the backup directory to S3", func() {
				Eventually(func() bool {
					return client.RemotePathExistsInBucket(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
				}, time.Minute).Should(BeTrue())
			})
		})

		Context("manual backup", func() {
			BeforeEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit stop service-backup")
			})

			AfterEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit start service-backup")
			})

			It("uploads files in the backup directory", func() {
				boshSSH("sudo /var/vcap/jobs/service-backup/bin/manual-backup")
				Eventually(func() bool {
					return client.RemotePathExistsInBucket(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
				}, time.Minute).Should(BeTrue())
			})
		})
	})

	Context("backing up to S3 with deployment name in path", func() {
		var (
			client *s3testclient.S3TestClient
		)

		BeforeEach(func() {
			boshManifest = envMustHave("S3_WITH_EXTENDED_PATH_BOSH_MANIFEST")

			awsAccessKeyID := envMustHave("AWS_ACCESS_KEY_ID")
			awsSecretKey := envMustHave("AWS_SECRET_ACCESS_KEY")
			client = s3testclient.New("https://s3.amazonaws.com", awsAccessKeyID, awsSecretKey, testPath)
		})

		AfterEach(func() {
			boshSSH(fmt.Sprintf("rm -f %s%s", testSourceFolder, toBackup))
			Expect(client.DeleteRemotePath(bucketName, testPath, "")).To(Succeed())
		})

		Context("manual backup", func() {
			BeforeEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit stop service-backup")
			})

			AfterEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit start service-backup")
			})

			It("uploads files in the backup directory", func() {
				boshSSH("sudo /var/vcap/jobs/service-backup/bin/manual-backup")
				Eventually(func() bool {
					return client.RemotePathExistsInBucket(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath+"/service-backup-ci-s3-extended-path"), toBackup))
				}, time.Minute).Should(BeTrue())
			})
		})
	})

	Context("backing up to Azure", func() {
		var (
			azureBlobService storage.BlobStorageClient
		)

		BeforeEach(func() {
			boshManifest = envMustHave("AZURE_BOSH_MANIFEST")
			azureAccountName := envMustHave("AZURE_STORAGE_ACCOUNT")
			azureAccountKey := envMustHave("AZURE_STORAGE_ACCESS_KEY")
			azureClient, err := storage.NewBasicClient(azureAccountName, azureAccountKey)
			Expect(err).ToNot(HaveOccurred())
			azureBlobService = azureClient.GetBlobService()
		})

		AfterEach(func() {
			boshSSH(fmt.Sprintf("rm -f %s%s", testSourceFolder, toBackup))

			_, err := azureBlobService.DeleteBlobIfExists(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
			Expect(err).NotTo(HaveOccurred())
		})

		Context("automatic backup", func() {
			It("Uploads files in the backup directory", func() {
				Eventually(func() bool {
					exists, err := azureBlobService.BlobExists(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
					Expect(err).NotTo(HaveOccurred())
					return exists
				}, time.Minute).Should(BeTrue())
			})
		})

		Context("manual backup", func() {
			BeforeEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit stop service-backup")
			})

			AfterEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit start service-backup")
			})

			It("uploads files in the backup directory", func() {
				boshSSH("sudo /var/vcap/jobs/service-backup/bin/manual-backup")
				Eventually(func() bool {
					exists, err := azureBlobService.BlobExists(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
					Expect(err).NotTo(HaveOccurred())
					return exists
				}, time.Minute).Should(BeTrue())
			})
		})
	})

	Context("backing up to SCP", func() {
		BeforeEach(func() {
			boshManifest = envMustHave("SCP_BOSH_MANIFEST")
			publicKey := envMustHave("SCP_PUBLIC_KEY")

			boshSSH(fmt.Sprintf("sudo mkdir -p ~vcap/.ssh && echo %s | sudo tee -a ~vcap/.ssh/authorized_keys", publicKey))
		})

		AfterEach(func() {
			boshSSH(fmt.Sprintf("rm -f %s%s", testSourceFolder, toBackup))
		})

		Context("automatic backup", func() {
			It("Uploads files in the backup directory", func() {
				Eventually(func() bool {
					return strings.Contains(boshSSH("sudo find /home/vcap/backups -type f"), toBackup)
				}, time.Minute).Should(BeTrue())
			})
		})

		Context("manual backup", func() {
			BeforeEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit stop service-backup")
			})

			AfterEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit start service-backup")
			})

			It("Uploads files in the backup directory", func() {
				boshSSH("sudo /var/vcap/jobs/service-backup/bin/manual-backup")
				Eventually(func() bool {
					return strings.Contains(boshSSH("sudo find /home/vcap/backups -type f"), toBackup)
				}, time.Minute).Should(BeTrue())
			})
		})
	})

	Context("backing up to Google Cloud Storage", func() {
		var (
			ctx                       context.Context
			bucket                    *gcs.BucketHandle
			gcpServiceAccountFilePath string
		)

		BeforeEach(func() {
			boshManifest = envMustHave("GCS_BOSH_MANIFEST")

			var manifest struct {
				InstanceGroups []struct {
					Jobs []struct {
						Properties struct {
							ServiceBackup struct {
								Destinations []struct {
									Config struct {
										ServiceAccountJSON string `yaml:"service_account_json"`
										BucketName         string `yaml:"bucket_name"`
									} `yaml:"config"`
								} `yaml:"destinations"`
							} `yaml:"service-backup"`
						} `yaml:"properties"`
					} `yaml:"jobs"`
				} `yaml:"instance_groups"`
			}
			manifestBytes, err := ioutil.ReadFile(boshManifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml.Unmarshal(manifestBytes, &manifest)).To(Succeed())

			gcpServiceAccountFile, err := ioutil.TempFile("", "service-backup-system-tests")
			Expect(err).NotTo(HaveOccurred())
			defer gcpServiceAccountFile.Close()
			gcpServiceAccountFilePath = gcpServiceAccountFile.Name()
			_, err = gcpServiceAccountFile.WriteString(manifest.InstanceGroups[0].Jobs[0].Properties.ServiceBackup.Destinations[0].Config.ServiceAccountJSON)
			Expect(err).NotTo(HaveOccurred())

			ctx = context.Background()
			gcsClient, err := gcs.NewClient(ctx, option.WithServiceAccountFile(gcpServiceAccountFile.Name()))
			Expect(err).NotTo(HaveOccurred())
			bucketName := manifest.InstanceGroups[0].Jobs[0].Properties.ServiceBackup.Destinations[0].Config.BucketName
			bucket = gcsClient.Bucket(bucketName)
		})

		AfterEach(func() {
			boshSSH(fmt.Sprintf("rm -f %s%s", testSourceFolder, toBackup))
			testhelpers.DeleteGCSBucket(ctx, bucket)
			Expect(os.Remove(gcpServiceAccountFilePath)).To(Succeed())
		})

		errorUploadingToGCS := func() error {
			today := time.Now()
			path := fmt.Sprintf("%d/%02d/%02d/%s", today.Year(), today.Month(), today.Day(), toBackup)
			gcsObject, err := bucket.Object(path).NewReader(ctx)
			if err != nil {
				return err
			}
			defer gcsObject.Close()
			content, err := ioutil.ReadAll(gcsObject)
			if err != nil {
				return err
			}
			if string(content) != "This should end up on S3\n" {
				return fmt.Errorf("file content was unexpected: '%s'", string(content))
			}
			return nil
		}

		Context("automatic backup", func() {
			It("uploads files in the backup directory", func() {
				Eventually(errorUploadingToGCS, time.Second*20).ShouldNot(HaveOccurred())
			})
		})

		Context("manual backup", func() {
			BeforeEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit stop service-backup")
			})

			AfterEach(func() {
				boshSSH("sudo /var/vcap/bosh/bin/monit start service-backup")
			})

			It("uploads files in the backup directory", func() {
				boshSSH("sudo /var/vcap/jobs/service-backup/bin/manual-backup")
				Eventually(errorUploadingToGCS, time.Second*20).ShouldNot(HaveOccurred())
			})
		})
	})
})

func envMustHave(key string) string {
	value := os.Getenv(key)
	Expect(value).ToNot(BeEmpty(), key)
	return value
}

func pathWithDate(path string) string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return path + "/" + datePath
}
