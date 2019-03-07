package azure_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"time"

	"path/filepath"

	"io/ioutil"
	"os"

	"bytes"

	"io"

	"code.cloudfoundry.org/lager"
	. "github.com/pivotal-cf/service-backup/azure"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/process/fakes"
)

var _ = Describe("Azure backup", func() {
	Describe("Upload", func() {
		var (
			processManager *fakes.FakeProcessManager
			name           string
			accountKey     string
			accountName    string
			container      string
			command        string
			localPath      string
			logger         lager.Logger
			remotePathFn   func() string
			endpoint       string
		)

		BeforeEach(func() {
			processManager = new(fakes.FakeProcessManager)
			name = "name"
			accountKey = "accountKey"
			accountName = "accountName"
			container = "container"
			command = "blobxfer"
			remotePathFn = func() string { return "" }
			localPath = "/tmp"
			logger = lager.NewLogger("test")
		})

		Context("without endpoint configured", func() {
			It("doesn't add --endpoint to blobxfer", func() {
				azureClient := New(name, accountKey, accountName, container, endpoint, command, remotePathFn)
				err := azureClient.Upload(localPath, logger, processManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(processManager.StartCallCount()).To(Equal(1))
				startCmd := processManager.StartArgsForCall(0)
				Expect(startCmd.Args).To(
					Equal([]string{command, "upload", "--local-path", localPath, "--remote-path", container, "--storage-account", accountName}),
				)
			})
		})

		Context("with endpoint configured", func() {
			It("passes --endpoint to blobxfer", func() {
				endpoint = "a.new.endpoint"
				azureClient := New(name, accountKey, accountName, container, endpoint, command, remotePathFn)
				err := azureClient.Upload(localPath, logger, processManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(processManager.StartCallCount()).To(Equal(1))
				startCmd := processManager.StartArgsForCall(0)
				Expect(startCmd.Args).To(
					Equal(
						[]string{command, "upload", "--local-path", localPath, "--remote-path", container, "--storage-account", accountName, "--endpoint", endpoint},
					),
				)
			})
		})
	})

	Context("when the child Azure process fails", func() {
		It("reports the failure", func() {
			fakeAzureCmd := assetPath("fail_with_output")
			fakeRemotePathFn := func() string { return "hi" }
			azureClient := New("foo", "foo", "foo", "foo", "foo", fakeAzureCmd, fakeRemotePathFn)
			bytes, logger := newFakeLogger()

			uploadErr := azureClient.Upload("/tmp", logger, process.NewManager())

			logOutput, err := ioutil.ReadAll(bytes)
			Expect(err).NotTo(HaveOccurred())

			Expect(uploadErr).To(HaveOccurred())
			Expect(string(logOutput)).To(ContainSubstring("This is a fake failure!"))
		})
	})

	It("when the process manager gets the terminate call, the child azure upload process gets a sigterm", func() {
		fakeAzureCmd := assetPath("term_trapper")
		fakeRemotePathFn := func() string { return "hi" }

		evidenceFile, err := ioutil.TempFile("", "azure-test")
		Expect(err).ToNot(HaveOccurred())
		evidencePath := evidenceFile.Name()
		err = os.Remove(evidencePath)
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(evidencePath)

		processManager := process.NewManager()

		// using accountName field to inject evidence file path to fake upload executable
		azureClient := New("foo", "foo", evidencePath, "foo", "foo", fakeAzureCmd, fakeRemotePathFn)

		go func() {
			azureClient.Upload("/tmp", lager.NewLogger("foo"), processManager)
		}()

		time.Sleep(100 * time.Millisecond)
		processManager.Terminate()
		SetDefaultEventuallyTimeout(2 * time.Second)
		Eventually(evidencePath).Should(BeAnExistingFile())
	})
})

func assetPath(filename string) string {
	path, err := filepath.Abs(filepath.Join("assets", filename))
	Expect(err).ToNot(HaveOccurred())
	return path
}

func newFakeLogger() (io.ReadWriter, lager.Logger) {
	b := new(bytes.Buffer)
	bSink := lager.NewWriterSink(b, lager.DEBUG)
	logger := lager.NewLogger("")
	logger.RegisterSink(bSink)

	return b, logger
}
