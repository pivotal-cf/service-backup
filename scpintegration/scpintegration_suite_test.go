package scpintegration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const (
	sshKeyUsername = "service-backup-tmp-key"
)

type TestData struct {
	PathToServiceBackupBinary string
	PrivateKeyPath            string
	UnixUser                  *user.User
}

var (
	pathToServiceBackupBinary string
	privateKeyPath            string
	privateKeyContents        []byte
	unixUser                  *user.User
)

func createSSHKey() (string, string) {
	sshKeys, err := ioutil.TempDir("", "scp-unit-tests")
	Expect(err).ToNot(HaveOccurred())
	privateKeyPath = filepath.Join(sshKeys, "id_rsa")
	Expect(exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-C", sshKeyUsername,
		"-N", "", "-f", privateKeyPath).Run()).To(Succeed())
	privateKeyContents, err = ioutil.ReadFile(privateKeyPath)
	Expect(err).ToNot(HaveOccurred())
	return filepath.Join(sshKeys, "id_rsa.pub"), privateKeyPath
}

func addToAuthorizedKeys(publicKeyPath string) {
	Expect(os.MkdirAll(filepath.Join(unixUser.HomeDir, ".ssh"), 0700)).To(Succeed())
	authKeys, err := os.OpenFile(filepath.Join(unixUser.HomeDir, ".ssh", "authorized_keys"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	Expect(err).ToNot(HaveOccurred())
	pubKey, err := os.Open(publicKeyPath)
	Expect(err).ToNot(HaveOccurred())
	defer authKeys.Close()
	defer pubKey.Close()
	_, err = io.Copy(authKeys, pubKey)
	Expect(err).ToNot(HaveOccurred())
}

func removeKeyFromAuthorized() {
	authKeysFilePath := filepath.Join(unixUser.HomeDir, ".ssh", "authorized_keys")
	authKeysContent, err := ioutil.ReadFile(authKeysFilePath)
	Expect(err).NotTo(HaveOccurred())

	trimmedAuthKeysLines := [][]byte{}
	for _, line := range bytes.Split(authKeysContent, []byte("\n")) {
		if !bytes.Contains(line, []byte(sshKeyUsername)) {
			trimmedAuthKeysLines = append(trimmedAuthKeysLines, line)
		}
	}

	trimmedAuthKeysContent := bytes.Join(trimmedAuthKeysLines, []byte("\n"))
	err = ioutil.WriteFile(authKeysFilePath, trimmedAuthKeysContent, 0600)
	Expect(err).NotTo(HaveOccurred())
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var publicKeyPath string
	publicKeyPath, privateKeyPath = createSSHKey()

	var err error
	unixUser, err = user.Current()
	Expect(err).NotTo(HaveOccurred())

	addToAuthorizedKeys(publicKeyPath)

	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf/service-backup")
	Expect(err).ToNot(HaveOccurred())

	forOtherNodes, err := json.Marshal(TestData{
		PathToServiceBackupBinary: pathToServiceBackupBinary,
		PrivateKeyPath:            privateKeyPath,
		UnixUser:                  unixUser,
	})
	Expect(err).ToNot(HaveOccurred())
	return forOtherNodes
}, func(data []byte) {
	var t TestData
	Expect(json.Unmarshal(data, &t)).To(Succeed())

	pathToServiceBackupBinary = t.PathToServiceBackupBinary
	privateKeyPath = t.PrivateKeyPath
	unixUser = t.UnixUser
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	removeKeyFromAuthorized()
	Expect(os.RemoveAll(filepath.Dir(privateKeyPath))).To(Succeed())

	gexec.CleanupBuildArtifacts()
})

func TestScpintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SCP Suite")
}
