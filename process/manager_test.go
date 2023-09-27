package process_test

import (
	"os/exec"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/process"
)

func alive(c *exec.Cmd) bool {
	return c.Process != nil &&
		c.Process.Signal(syscall.Signal(0)) == nil
}

var _ = Describe("process manager", func() {
	It("starts and terminates processes", func() {
		pt := process.NewManager()
		var commands []*exec.Cmd
		for i := 0; i < 25; i++ {
			cmd := exec.Command("sleep", "42")
			commands = append(commands, cmd)
			go func() { pt.Start(cmd) }()
		}

		pt.Terminate()

		for _, c := range commands {
			Eventually(alive(c)).Should(BeFalse())
		}
	})

	It("can perform two consecutive non-overlapping starts", func() {
		pt := process.NewManager()
		cmd1 := exec.Command("true")
		cmd2 := exec.Command("true")

		pt.Start(cmd1)
		_, err := pt.Start(cmd2)
		Expect(err).NotTo(HaveOccurred())

		pt.Terminate()

		Expect(alive(cmd1)).To(BeFalse())
		Expect(alive(cmd2)).To(BeFalse())
	})

	It("can perform two consecutive overlapping starts", func() {
		pt := process.NewManager()
		cmd1 := exec.Command("sleep", "0.1")
		cmd2 := exec.Command("sleep", "0.3")
		cmd3 := exec.Command("sleep", "0.1")

		cmd1Started := make(chan bool, 1)
		cmd1Done := make(chan bool, 1)
		cmd3Finished := make(chan bool, 1)

		go func() {
			cmd1Started <- true
			pt.Start(cmd1)
			cmd1Done <- true
		}()
		go func() {
			<-cmd1Started
			pt.Start(cmd2)
		}()
		go func() {
			<-cmd1Done
			_, err := pt.Start(cmd3)
			Expect(err).NotTo(HaveOccurred())
			cmd3Finished <- true
		}()

		<-cmd3Finished
		pt.Terminate()

		Expect(alive(cmd1)).To(BeFalse())
		Expect(alive(cmd2)).To(BeFalse())
		Expect(alive(cmd3)).To(BeFalse())
	})

	It("produces error if executable doesn't exist", func() {
		pt := process.NewManager()
		cmd1 := exec.Command("idonotexist123")
		cmd2 := exec.Command("idonotexist124")

		_, err := pt.Start(cmd1)
		Expect(err).To(HaveOccurred())

		_, err = pt.Start(cmd2)
		Expect(err).To(HaveOccurred())
	})

	It("produces error if executable has nonzero exit", func() {
		pt := process.NewManager()
		cmd := exec.Command("false")

		_, err := pt.Start(cmd)
		Expect(err).To(HaveOccurred())
	})

	It("doesn't kill self and the test runner if never given anything to start", func() {
		pt := process.NewManager()
		pt.Terminate()
		Expect(true).To(BeTrue())
	})

	It("captures stdout from the executable", func() {
		const minimalBlockingByteCount = 128*1024 + 1
		cmdstr := "for i in $(seq 1 131073); do echo -n X; done"
		pt := process.NewManager()
		cmd := exec.Command("bash", "-c", cmdstr)

		var (
			out   []byte
			errCh = make(chan error, 1)
		)

		go func() {
			var err error
			out, err = pt.Start(cmd)
			errCh <- err
		}()

		select {
		case err := <-errCh:
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).Should(Equal(strings.Repeat("X", minimalBlockingByteCount)))
		case <-time.After(5 * time.Second):
			Fail("Expected command to exit within 5s, but it did not.")
		}
	})

	It("captures stderr from the executable", func() {
		pt := process.NewManager()
		cmd := exec.Command("rm", "foobar")

		out, _ := pt.Start(cmd)
		Expect(string(out)).Should(ContainSubstring("No such file or directory"))
	})
})
