package upload

import (
	"errors"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Locator", func() {
	DescribeTable("locates certificates",
		func(caCertOnDisk string, errMatcher types.GomegaMatcher) {
			statCallCount := 0
			stat = func(name string) (os.FileInfo, error) {
				statCallCount += 1

				if caCertOnDisk == name {
					return nil, nil
				}

				return nil, &os.PathError{"stat", name, errors.New("file not found")}
			}

			found, err := CACertPath()

			Expect(found).To(Equal(caCertOnDisk))
			Expect(err).To(errMatcher)
			Expect(statCallCount).To(BeNumerically(">", 0))
		},
		Entry("Debian/Ubuntu/Gentoo etc.", "/etc/ssl/certs/ca-certificates.crt", BeNil()),
		Entry("Fedora/RHEL", "/etc/pki/tls/certs/ca-bundle.crt", BeNil()),
		Entry("OpenSUSE", "/etc/ssl/ca-bundle.pem", BeNil()),
		Entry("OpenELEC", "/etc/pki/tls/cacert.pem", BeNil()),
		Entry("Non-linux OS", "", MatchError("could not locate a known ca cert")),
	)
})
