// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

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
