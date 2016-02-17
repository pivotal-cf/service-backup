package azure

import (
	"io/ioutil"
	"os"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/pivotal-golang/lager"
)

type AzureClient struct {
	accountName string
	accountKey  string
	container   string
	logger      lager.Logger
}

func New(accountKey, accountName, container string, logger lager.Logger) *AzureClient {
	return &AzureClient{accountKey: accountKey, accountName: accountName, container: container, logger: logger}
}

// Azure doesn't have directories, so RemotePathExists and CreateRemotePath are no-ops

func (*AzureClient) RemotePathExists(remotePath string) (bool, error) {
	return true, nil
}

func (*AzureClient) CreateRemotePath(remotePath string) error {
	return nil
}

func (a *AzureClient) Upload(localPath, remotePath string) error {
	a.logger.Info("Creating Azure client", lager.Data{"accountName": a.accountName})
	azureClient, err := storage.NewBasicClient(a.accountName, a.accountKey)
	if err != nil {
		return err
	}
	azureBlobService := azureClient.GetBlobService()

	a.logger.Info("Uploading blobs", lager.Data{"localPath": localPath, "remotePath": remotePath})
	return a.uploadDirectory(azureBlobService, localPath, remotePath)
}

func (a *AzureClient) uploadDirectory(azureBlobService storage.BlobStorageClient, localDirPath, remoteDirPath string) error {
	localFiles, err := ioutil.ReadDir(localDirPath)
	if err != nil {
		return err
	}

	for _, localFile := range localFiles {
		fileName := localFile.Name()
		localFilePath := localDirPath + "/" + fileName
		remoteFilePath := remoteDirPath + "/" + fileName

		if localFile.IsDir() {
			err = a.uploadDirectory(azureBlobService, localFilePath, remoteFilePath)
		} else {
			length := uint64(localFile.Size())
			err = a.uploadFile(azureBlobService, localFilePath, remoteFilePath, length)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *AzureClient) uploadFile(azureBlobService storage.BlobStorageClient, localFilePath, remoteFilePath string, length uint64) error {
	file, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	a.logger.Info("Uploading blob", lager.Data{"localPath": localFilePath, "remotePath": remoteFilePath, "length": length})
	return azureBlobService.CreateBlockBlobFromReader(a.container, remoteFilePath, length, file, nil)
}
