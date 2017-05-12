package gcs_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGCS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Google Cloud Storage Suite")
}
