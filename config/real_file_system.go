package config

import "os"

type RealFileSystem struct{}

func (f RealFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
