package scp

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-golang/lager"
)

type SCPClient struct {
	host       string
	port       int
	username   string
	privateKey string
	basePath   string
}

func New(host string, port int, username, privateKeyPath, basePath string, logger lager.Logger) *SCPClient {
	return &SCPClient{
		host:       host,
		port:       port,
		username:   username,
		privateKey: privateKeyPath,
		basePath:   basePath,
	}
}

func (client *SCPClient) generateBackupKey() (string, error) {
	privateKeyFile, err := ioutil.TempFile("", "backup_key")
	if err != nil {
		return "", err
	}
	privateKeyFile.WriteString(client.privateKey)
	privateKeyFile.Close()
	privateKeyFile.Chmod(0400)
	return privateKeyFile.Name(), nil
}

func (client *SCPClient) generateKnownHosts(sessionLogger lager.Logger) (string, error) {
	cmd := exec.Command("ssh-keyscan", "-p", strconv.Itoa(client.port), client.host)
	sshKeyscanOutput, err := cmd.CombinedOutput()
	if err != nil {
		wrappedErr := fmt.Errorf("error performing ssh-keyscan: '%s', output: '%s'", err, sshKeyscanOutput)
		sessionLogger.Error("scp", wrappedErr)
		return "", wrappedErr
	}
	knownHostsFile, err := ioutil.TempFile("", "known_hosts")
	if err != nil {
		return "", err
	}
	knownHostsFile.WriteString(string(sshKeyscanOutput))
	knownHostsFile.Close()
	return knownHostsFile.Name(), nil
}

func (client *SCPClient) Upload(localPath string, sessionLogger lager.Logger) error {
	privateKeyFileName, err := client.generateBackupKey()
	if err != nil {
		return err
	}

	knownHostsFileName, err := client.generateKnownHosts(sessionLogger)
	if err != nil {
		return err
	}

	defer os.Remove(privateKeyFileName)

	remotePathGenerator := backup.RemotePathGenerator{}
	remotePath := remotePathGenerator.RemotePathWithDate(client.basePath)

	if err := client.ensureRemoteDirectoryExists(remotePath, privateKeyFileName, knownHostsFileName, sessionLogger); err != nil {
		return err
	}

	scpDest := fmt.Sprintf("%s@%s:%s", client.username, client.host, remotePath)
	cmd := exec.Command("scp", "-i", privateKeyFileName, "-oUserKnownHostsFile="+knownHostsFileName, "-P", strconv.Itoa(client.port), "-r", ".", scpDest)

	cmd.Dir = localPath
	scpCommandOutput, err := cmd.CombinedOutput()
	if err != nil {
		wrappedErr := fmt.Errorf("error performing SCP: '%s', output: '%s'", err, scpCommandOutput)
		sessionLogger.Error("scp", wrappedErr)
		return wrappedErr
	}

	sessionLogger.Info("scp completed")
	return nil
}

func (client *SCPClient) ensureRemoteDirectoryExists(remotePath, privateKeyFileName, knownHostsFileName string, sessionLogger lager.Logger) error {
	cmd := exec.Command("ssh", "-i", privateKeyFileName, "-oUserKnownHostsFile="+knownHostsFileName, "-p", fmt.Sprintf("%d", client.port),
		fmt.Sprintf("%s@%s", client.username, client.host),
		fmt.Sprintf("mkdir -p %s", remotePath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		wrappedErr := fmt.Errorf("error checking if remote path exists: '%s', output: '%s'", err, output)
		sessionLogger.Error("ssh", wrappedErr)
		return wrappedErr
	}

	return nil
}
