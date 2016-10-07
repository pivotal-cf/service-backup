package gcp_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGcp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Google Cloud Storage Suite")
}
