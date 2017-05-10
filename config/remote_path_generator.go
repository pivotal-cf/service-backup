package config

import (
	"fmt"
	"time"
)

type RemotePathGenerator struct {
	BasePath string
}

func (r RemotePathGenerator) RemotePathWithDate() string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())

	if r.BasePath == "" {
		return datePath
	}
	return r.BasePath + "/" + datePath
}
