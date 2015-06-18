package main

import (
	"io/ioutil"
	"os"
	"os/exec"
)

func main() {
	args := os.Args[1:]
	cmd := exec.Command(
		"/var/vcap/packages/aws-cli/bin/aws",
		"s3",
		"cp",
		args[0],
		args[1],
		"--endpoint-url",
		args[2],
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = ioutil.WriteFile(args[3], []byte(err.Error()), 0644)

	}
	if out != nil {
		ioutil.WriteFile(args[3], out, 0644)
	}
}
