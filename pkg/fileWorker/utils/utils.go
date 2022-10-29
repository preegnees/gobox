package utils

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func GetModTime(printErr func (string, string, error) error, path string) (int64, error) {

	f, err := os.Stat(path)
	if err != nil {
		return -1, printErr("getModTime() err os Stat, event", path, err)
	}

	return f.ModTime().UTC().UnixMicro(), nil
}

func IsFolder(printErr func (string, string, error) error, log logrus.Logger, path string) (bool, error) {

	log.Debug(fmt.Sprintf("[watcher] isFolder(): %s", path))

	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, printErr("isFolder(), path", path, err)
	}

	return fileInfo.IsDir(), nil
}