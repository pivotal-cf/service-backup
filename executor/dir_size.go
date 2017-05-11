package executor

import (
	"os"
	"path/filepath"
)

func calculateDirSize(localPath string) (int64, error) {
	var size int64
	var err error
	if _, err = os.Stat(localPath); err == nil {
		err = filepath.Walk(localPath, func(_ string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				size += info.Size()
			}
			return err
		})
	}
	return size, err
}
