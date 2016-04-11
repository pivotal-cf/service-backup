package scp

import (
	"fmt"
	"os/exec"

	"github.com/pivotal-golang/lager"
)

type SCPClient struct {
	host           string
	port           int
	username       string
	privateKeyPath string
	logger         lager.Logger
	sessionLogger  lager.Logger
}

func New(host string, port int, username, privateKeyPath string, logger lager.Logger) *SCPClient {
	return &SCPClient{
		host:           host,
		port:           port,
		username:       username,
		privateKeyPath: privateKeyPath,
		logger:         logger,
		sessionLogger:  logger,
	}
}

func (client *SCPClient) Upload(localPath, remotePath string) error {
	if err := client.ensureRemoteDirectoryExists(remotePath); err != nil {
		return err
	}

	scpDest := fmt.Sprintf("%s@%s:%s", client.username, client.host, remotePath)
	cmd := exec.Command("scp", "-i", client.privateKeyPath, "-P", fmt.Sprintf("%d", client.port), "-r", ".", scpDest)
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

func (client *SCPClient) ensureRemoteDirectoryExists(remotePath string) error {
	cmd := exec.Command("ssh", "-i", client.privateKeyPath, "-p", fmt.Sprintf("%d", client.port),
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
