package release_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestReleaseTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ReleaseTests Suite")
}
