package azure

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pivotal-golang/lager"
)

type AzureClient struct {
	accountName      string
	accountKey       string
	container        string
	blobStoreBaseUrl string
	azureCmd         string
	logger           lager.Logger
}

func New(accountKey, accountName, container, blobStoreBaseUrl, azureCmd string, logger lager.Logger) *AzureClient {
	return &AzureClient{accountKey: accountKey, accountName: accountName, container: container, blobStoreBaseUrl: blobStoreBaseUrl, logger: logger, azureCmd: azureCmd}
}

func (a *AzureClient) Upload(localPath, remotePath string) error {
	a.logger.Info("Uploading azure blobs", lager.Data{"container": a.container, "localPath": localPath, "remotePath": remotePath})
	a.logger.Info("The container and remote path will be created if they don't already exist", lager.Data{"container": a.container, "remotePath": remotePath})
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

	a.logger.Info("Uploading blob", lager.Data{"localPath": localFilePath, "remotePath": remoteFilePath, "length": length})
	return exec.Command(a.azureCmd, fmt.Sprintf("--storageaccountkey=%s", a.accountKey), fmt.Sprintf("--remoteresource=%s", remoteFilePath), a.accountName, a.container, localFilePath).Run()
}
