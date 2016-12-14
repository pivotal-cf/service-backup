package systemtruststorelocator_test

import (
	"errors"
	"os"

	"github.com/pivotal-cf-experimental/service-backup/systemtruststorelocator"
	"github.com/pivotal-cf-experimental/service-backup/systemtruststorelocator/locatorfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Locator", func() {
	var (
		locator        systemtruststorelocator.Locator
		fileSystem     *locatorfakes.FakeFileSystem
		fileThatExists string
	)

	BeforeEach(func() {
		fileSystem = new(locatorfakes.FakeFileSystem)
	})

	JustBeforeEach(func() {
		fileSystem.StatStub = func(name string) (os.FileInfo, error) {
			if name == fileThatExists {
				return nil, nil
			}
			return nil, &os.PathError{"stat", name, errors.New("file not found")}
		}

		locator = systemtruststorelocator.New(fileSystem)
	})

	AfterEach(func() {
		Expect(fileSystem.StatCallCount()).To(BeNumerically(">=", 1))
	})

	Context("when the system is Debian/Ubuntu/Gentoo etc.", func() {
		BeforeEach(func() {
			fileThatExists = "/etc/ssl/certs/ca-certificates.crt"
		})

		It("knows the path to the system trust store", func() {
			Expect(locator.Path()).To(Equal("/etc/ssl/certs/ca-certificates.crt"))
		})
	})

	Context("when the system is Fedora/RHEL", func() {
		BeforeEach(func() {
			fileThatExists = "/etc/pki/tls/certs/ca-bundle.crt"
		})

		It("knows the path to the system trust store", func() {
			Expect(locator.Path()).To(Equal("/etc/pki/tls/certs/ca-bundle.crt"))
		})
	})

	Context("when the system is OpenSUSE", func() {
		BeforeEach(func() {
			fileThatExists = "/etc/ssl/ca-bundle.pem"
		})

		It("knows the path to the system trust store", func() {
			Expect(locator.Path()).To(Equal("/etc/ssl/ca-bundle.pem"))
		})
	})

	Context("when the system is OpenELEC", func() {
		BeforeEach(func() {
			fileThatExists = "/etc/pki/tls/cacert.pem"
		})

		It("knows the path to the system trust store", func() {
			Expect(locator.Path()).To(Equal("/etc/pki/tls/cacert.pem"))
		})
	})

	Context("when the system is not Linux", func() {
		BeforeEach(func() {
			fileThatExists = "not/a/path/to/a/linux/system/trust/store"
		})

		It("returns an error", func() {
			_, err := locator.Path()
			Expect(err).To(MatchError("could not locate system trust store"))
		})
	})
})
