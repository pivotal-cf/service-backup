package processterminator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestProcessterminator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Processterminator Suite")
}
