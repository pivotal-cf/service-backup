package scp

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"code.cloudfoundry.org/lager"
)

type SCPClient struct {
	name         string
	host         string
	port         int
	username     string
	privateKey   string
	fingerPrint  string
	remotePathFn func() string
}

func New(name, host string, port int, username, privateKeyPath, fingerPrint string, remotePathFn func() string) *SCPClient {
	return &SCPClient{
		name:         name,
		host:         host,
		port:         port,
		username:     username,
		privateKey:   privateKeyPath,
		fingerPrint:  fingerPrint,
		remotePathFn: remotePathFn,
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
	knownHostsContent := ""
	if client.fingerPrint == "" {
		sessionLogger.Info("Fingerprint not found, performing key-scan")
		cmd := exec.Command("ssh-keyscan", "-p", strconv.Itoa(client.port), client.host)
		sshKeyscanOutput, err := cmd.CombinedOutput()
		if err != nil {
			wrappedErr := fmt.Errorf("error performing ssh-keyscan: '%s', output: '%s'", err, sshKeyscanOutput)
			sessionLogger.Error("scp", wrappedErr)
			return "", wrappedErr
		}
		knownHostsContent = string(sshKeyscanOutput)
	} else {
		knownHostsContent = client.fingerPrint
	}

	knownHostsFile, err := ioutil.TempFile("", "known_hosts")
	if err != nil {
		return "", err
	}
	knownHostsFile.WriteString(knownHostsContent)
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

	remotePath := client.remotePathFn()

	if err = client.ensureRemoteDirectoryExists(remotePath, privateKeyFileName, knownHostsFileName, sessionLogger); err != nil {
		return err
	}

	scpDest := fmt.Sprintf("%s@%s:%s", client.username, client.host, remotePath)
	cmd := exec.Command("scp", "-oStrictHostKeyChecking=yes", "-i", privateKeyFileName, "-oUserKnownHostsFile="+knownHostsFileName, "-P", strconv.Itoa(client.port), "-r", ".", scpDest)

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
	cmd := exec.Command("ssh", "-oStrictHostKeyChecking=yes", "-i", privateKeyFileName, "-oUserKnownHostsFile="+knownHostsFileName, "-p", fmt.Sprintf("%d", client.port),
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

func (c *SCPClient) Name() string {
	return c.name
}
