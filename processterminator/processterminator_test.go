package processterminator_test

import (
	"os/exec"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/processterminator"
)

func alive(c *exec.Cmd) func() bool {
	return func() bool {
		return c.Process != nil &&
			c.Process.Signal(syscall.Signal(0)) == nil
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

		Eventually(alive(cmd1)).Should(BeTrue())
		Eventually(alive(cmd2)).Should(BeTrue())
		Eventually(alive(cmd3)).Should(BeTrue())

		pt.Terminate()

		Eventually(alive(cmd1)).Should(BeFalse())
		Eventually(alive(cmd2)).Should(BeFalse())
		Eventually(alive(cmd3)).Should(BeFalse())
	})

	It("works with a lot of commands", func() {
		pt := processterminator.New()
		var commands []*exec.Cmd
		for i := 0; i < 25; i++ {
			cmd := exec.Command("sleep", "42")
			commands = append(commands, cmd)
			go func() { pt.Start(cmd) }()
			Eventually(alive(cmd)).Should(BeTrue())
		}

		pt.Terminate()

		for _, c := range commands {
			Eventually(alive(c)).Should(BeFalse())
		}
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

	It("doesn't kill self and the test runner if never given anything to start", func() {
		pt := processterminator.New()
		pt.Terminate()
		Expect(true).To(BeTrue())
	})
})
