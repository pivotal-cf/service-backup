package upload

import (
	"errors"
	"os"
)

var stat = os.Stat

func CACertPath() (string, error) {
	// Source of paths to system trust store for Linux distributions:
	// https://golang.org/src/crypto/x509/root_linux.go
	certFiles := []string{
		"/etc/ssl/certs/ca-certificates.crt", // Debian/Ubuntu/Gentoo etc.
		"/etc/pki/tls/certs/ca-bundle.crt",   // Fedora/RHEL
		"/etc/ssl/ca-bundle.pem",             // OpenSUSE
		"/etc/pki/tls/cacert.pem",            // OpenELEC
	}

	for _, filePath := range certFiles {
		if _, err := stat(filePath); err == nil {
			return filePath, nil
		}
	}
	return "", errors.New("could not locate a known ca cert")
}
