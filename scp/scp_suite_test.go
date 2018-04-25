package scp_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestScp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scp Suite")
}
