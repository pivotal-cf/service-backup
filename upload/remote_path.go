package upload

import (
	"fmt"
	"time"
)

func RemotePathFunc(basePath, deploymentName string) func() string {
	return func() string {
		today := time.Now()
		datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())

		var path string
		if basePath != "" {
			path += basePath + "/"
		}

		if deploymentName != "" {
			path += deploymentName + "/"
		}

		return path + datePath
	}
}
