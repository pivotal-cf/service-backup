package scp

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"github.com/pivotal-golang/lager"
)

type SCPClient struct {
	host          string
	port          int
	username      string
	privateKey    string
	logger        lager.Logger
	sessionLogger lager.Logger
}

func New(host string, port int, username, privateKeyPath string, logger lager.Logger) *SCPClient {
	return &SCPClient{
		host:          host,
		port:          port,
		username:      username,
		privateKey:    privateKeyPath,
		logger:        logger,
		sessionLogger: logger,
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

func (client *SCPClient) generateKnownHosts() (string, error) {
	cmd := exec.Command("ssh-keyscan", "-p", strconv.Itoa(client.port), client.host)
	sshKeyscanOutput, err := cmd.CombinedOutput()
	if err != nil {
		wrappedErr := fmt.Errorf("error performing ssh-keyscan: '%s', output: '%s'", err, sshKeyscanOutput)
		client.sessionLogger.Error("scp", wrappedErr)
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

func (client *SCPClient) Upload(localPath, remotePath string) error {
	privateKeyFileName, err := client.generateBackupKey()
	if err != nil {
		return err
	}

	knownHostsFileName, err := client.generateKnownHosts()
	if err != nil {
		return err
	}

	defer os.Remove(privateKeyFileName)

	if err := client.ensureRemoteDirectoryExists(remotePath, privateKeyFileName, knownHostsFileName); err != nil {
		return err
	}

	scpDest := fmt.Sprintf("%s@%s:%s", client.username, client.host, remotePath)
	cmd := exec.Command("scp", "-i", privateKeyFileName, "-oUserKnownHostsFile="+knownHostsFileName, "-P", strconv.Itoa(client.port), "-r", ".", scpDest)

	cmd.Dir = localPath
	scpCommandOutput, err := cmd.CombinedOutput()
	if err != nil {
		wrappedErr := fmt.Errorf("error performing SCP: '%s', output: '%s'", err, scpCommandOutput)
		client.sessionLogger.Error("scp", wrappedErr)
		return wrappedErr
	}

	client.sessionLogger.Info("scp completed")
	return nil
}

func (client *SCPClient) ensureRemoteDirectoryExists(remotePath, privateKeyFileName, knownHostsFileName string) error {
	cmd := exec.Command("ssh", "-i", privateKeyFileName, "-oUserKnownHostsFile="+knownHostsFileName, "-p", fmt.Sprintf("%d", client.port),
		fmt.Sprintf("%s@%s", client.username, client.host),
		fmt.Sprintf("mkdir -p %s", remotePath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		wrappedErr := fmt.Errorf("error checking if remote path exists: '%s', output: '%s'", err, output)
		client.sessionLogger.Error("ssh", wrappedErr)
		return wrappedErr
	}

	return nil
}

//SetLogSession adds an identifier to all log messages for the duration of the session
func (client *SCPClient) SetLogSession(sessionName, sessionIdentifier string) {
	client.sessionLogger = client.logger.Session(
		sessionName,
		lager.Data{"identifier": sessionIdentifier},
	)
}

//CloseLogSession removes any previously added identifier from future log messages
func (client *SCPClient) CloseLogSession() {
	client.sessionLogger = client.logger
}
