package azure

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-golang/lager"
)

type AzureClient struct {
	name             string
	accountName      string
	accountKey       string
	container        string
	blobStoreBaseUrl string
	azureCmd         string
	basePath         string
}

func New(name, accountKey, accountName, container, blobStoreBaseUrl, azureCmd, basePath string) *AzureClient {
	return &AzureClient{
		name:             name,
		accountKey:       accountKey,
		accountName:      accountName,
		container:        container,
		blobStoreBaseUrl: blobStoreBaseUrl,
		basePath:         basePath,
		azureCmd:         azureCmd,
	}
}

func (a *AzureClient) Upload(localPath string, sessionLogger lager.Logger) error {
	remotePathGenerator := backup.RemotePathGenerator{}
	remotePath := remotePathGenerator.RemotePathWithDate(a.basePath)

	sessionLogger.Info("Uploading azure blobs", lager.Data{"container": a.container, "localPath": localPath, "remotePath": remotePath})
	sessionLogger.Info("The container and remote path will be created if they don't already exist", lager.Data{"container": a.container, "remotePath": remotePath})
	sessionLogger.Info(fmt.Sprintf("about to upload %s to Azure remote path %s", localPath, remotePath))
	return a.uploadDirectory(localPath, remotePath)
}

func (a *AzureClient) uploadDirectory(localDirPath, remoteDirPath string) error {
	localFiles, err := ioutil.ReadDir(localDirPath)
	if err != nil {
		return err
	}

	for _, localFile := range localFiles {
		fileName := localFile.Name()
		localFilePath := localDirPath + "/" + fileName
		remoteFilePath := remoteDirPath + "/" + fileName

		if localFile.IsDir() {
			err = a.uploadDirectory(localFilePath, remoteFilePath)
		} else {
			length := uint64(localFile.Size())
			err = a.uploadFile(localFilePath, remoteFilePath, length)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *AzureClient) uploadFile(localFilePath, remoteFilePath string, length uint64) error {
	file, err := os.Open(localFilePath)
	if err != nil {
		return err
	}

	defer file.Close()

	cmd := exec.Command(a.azureCmd, fmt.Sprintf("--remoteresource=%s", remoteFilePath), a.accountName, a.container, localFilePath)
	cmd.Env = append(cmd.Env, fmt.Sprintf("BLOBXFER_STORAGEACCOUNTKEY=%s", a.accountKey))
	return cmd.Run()
}

func (a *AzureClient) Name() string {
	return a.name
}
