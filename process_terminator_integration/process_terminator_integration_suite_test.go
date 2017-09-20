package process_terminator_integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestProcessTerminatorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ProcessTerminatorIntegration Suite")
}
