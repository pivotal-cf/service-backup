package systemtruststorelocator

import (
	"errors"
	"os"
)

// Source of paths to system trust store for Linux distributions:
// https://golang.org/src/crypto/x509/root_linux.go
var certFiles = []string{
	"/etc/ssl/certs/ca-certificates.crt", // Debian/Ubuntu/Gentoo etc.
	"/etc/pki/tls/certs/ca-bundle.crt",   // Fedora/RHEL
	"/etc/ssl/ca-bundle.pem",             // OpenSUSE
	"/etc/pki/tls/cacert.pem",            // OpenELEC
}

//go:generate counterfeiter -o locatorfakes/fake_file_system.go . FileSystem
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
}

type Locator struct {
	fileSystem FileSystem
}

func New(fileSystem FileSystem) Locator {
	return Locator{
		fileSystem: fileSystem,
	}
}

func (l Locator) Path() (string, error) {
	for _, filePath := range certFiles {
		if _, err := l.fileSystem.Stat(filePath); err == nil {
			return filePath, nil
		}
	}
	return "", errors.New("could not locate system trust store")
}
