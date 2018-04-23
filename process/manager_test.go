package process_test

import (
	"os/exec"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/process"
)

func alive(c *exec.Cmd) bool {
	return c.Process != nil &&
		c.Process.Signal(syscall.Signal(0)) == nil
}

var _ = Describe("process terminator", func() {
	It("starts and terminates processes", func() {
		pt := process.NewManager()
		var commands []*exec.Cmd
		for i := 0; i < 25; i++ {
			cmd := exec.Command("sleep", "42")
			started := make(chan struct{})
			commands = append(commands, cmd)
			go func() { pt.Start(cmd, started) }()
			Eventually(started).Should(BeClosed())
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

		pt.Start(cmd1, make(chan struct{}))
		err := pt.Start(cmd2, make(chan struct{}))
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
			pt.Start(cmd1, make(chan struct{}))
			cmd1Done <- true
		}()
		go func() {
			<-cmd1Started
			pt.Start(cmd2, make(chan struct{}))
		}()
		go func() {
			<-cmd1Done
			err := pt.Start(cmd3, make(chan struct{}))
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

		err := pt.Start(cmd1, make(chan struct{}))
		Expect(err).To(HaveOccurred())

		err = pt.Start(cmd2, make(chan struct{}))
		Expect(err).To(HaveOccurred())
	})

	It("produces error if executable has nonzero exit", func() {
		pt := process.NewManager()
		cmd := exec.Command("false")

		err := pt.Start(cmd, make(chan struct{}))
		Expect(err).To(HaveOccurred())
	})

	It("doesn't kill self and the test runner if never given anything to start", func() {
		pt := process.NewManager()
		pt.Terminate()
		Expect(true).To(BeTrue())
	})
})
