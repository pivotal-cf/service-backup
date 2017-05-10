package config

import (
	"fmt"
	"time"
)

type RemotePathGenerator struct {
	BasePath       string
	DeploymentName string
}

func (r RemotePathGenerator) RemotePathWithDate() string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())

	var path string
	if r.BasePath != "" {
		path += r.BasePath + "/"
	}

	if r.DeploymentName != "" {
		path += r.DeploymentName + "/"
	}

	return path + datePath
}
