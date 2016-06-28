package backup

import (
	"os"
	"path/filepath"
)

//go:generate counterfeiter -o backupfakes/fake_size_calculator.go . SizeCalculator
type SizeCalculator interface {
	DirSize(localPath string) (int64, error)
}

type FileSystemSizeCalculator struct{}

func (h *FileSystemSizeCalculator) DirSize(localPath string) (int64, error) {
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
