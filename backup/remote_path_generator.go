package backup

import (
	"fmt"
	"time"
)

type RemotePathGenerator struct{}

func (r *RemotePathGenerator) RemotePathWithDate(basePath string) string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return basePath + "/" + datePath
}
