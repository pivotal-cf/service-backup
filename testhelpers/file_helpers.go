package testhelpers

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/gomega"
)

func GetTempFilePath() string {
	f, err := ioutil.TempFile("", "process_manager")
	Expect(err).ToNot(HaveOccurred())
	err = os.Remove(f.Name())
	Expect(err).ToNot(HaveOccurred())
	return f.Name()
}
