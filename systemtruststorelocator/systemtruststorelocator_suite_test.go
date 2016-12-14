package systemtruststorelocator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSystemTrustStoreLocator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SystemTrustStoreLocator Suite")
}
