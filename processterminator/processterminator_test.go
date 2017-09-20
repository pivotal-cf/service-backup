package processterminator_test

import (
	"os/exec"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/processterminator"
)

func alive(c *exec.Cmd) bool {
	return c.Process != nil &&
		c.Process.Signal(syscall.Signal(0)) == nil
}

func aliveProbe(c *exec.Cmd) func() bool {
	return func() bool {
		return alive(c)
	}
}

var _ = Describe("process terminator", func() {
	It("starts and terminates processes", func() {
		pt := processterminator.New()
		cmd1 := exec.Command("sleep", "42")
		cmd2 := exec.Command("sleep", "42")
		cmd3 := exec.Command("sleep", "42")

		go func() { pt.Start(cmd1) }()
		go func() { pt.Start(cmd2) }()
		go func() { pt.Start(cmd3) }()

		Eventually(aliveProbe(cmd1)).Should(BeTrue())
		Eventually(aliveProbe(cmd2)).Should(BeTrue())
		Eventually(aliveProbe(cmd3)).Should(BeTrue())

		pt.Terminate()

		Expect(alive(cmd1)).To(BeFalse())
		Expect(alive(cmd2)).To(BeFalse())
		Expect(alive(cmd3)).To(BeFalse())
	})

	It("works with a lot of commands", func() {
		pt := processterminator.New()
		var commands []*exec.Cmd
		for i := 0; i < 25; i++ {
			cmd := exec.Command("sleep", "42")
			commands = append(commands, cmd)
			go func() { pt.Start(cmd) }()
			Eventually(aliveProbe(cmd)).Should(BeTrue())
		}

		pt.Terminate()

		for _, c := range commands {
			Eventually(alive(c)).Should(BeFalse())
		}
	})

	It("can perform two consecutive non-overlapping starts", func() {
		pt := processterminator.New()
		cmd1 := exec.Command("true")
		cmd2 := exec.Command("true")

		pt.Start(cmd1)
		err := pt.Start(cmd2)
		Expect(err).NotTo(HaveOccurred())

		pt.Terminate()

		Expect(alive(cmd1)).To(BeFalse())
		Expect(alive(cmd2)).To(BeFalse())
	})

	It("can perform two consecutive overlapping starts", func() {
		pt := processterminator.New()
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
			err := pt.Start(cmd3)
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
		pt := processterminator.New()
		cmd1 := exec.Command("idonotexist123")
		cmd2 := exec.Command("idonotexist124")

		err := pt.Start(cmd1)
		Expect(err).To(HaveOccurred())

		err = pt.Start(cmd2)
		Expect(err).To(HaveOccurred())
	})

	It("produces error if executable has nonzero exit", func() {
		pt := processterminator.New()
		cmd := exec.Command("false")

		err := pt.Start(cmd)
		Expect(err).To(HaveOccurred())
	})

	It("doesn't kill self and the test runner if never given anything to start", func() {
		pt := processterminator.New()
		pt.Terminate()
		Expect(true).To(BeTrue())
	})
})
