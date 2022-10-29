package utils

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/fsnotify/fsnotify"
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

type IClient interface {
	Send(Info) error
}

var (
	IGNORE_STRS = []string{"~", "__gobox__"}
)

// Info. Inforamtion about event
type Info struct {
	Action  fsnotify.Op
	Path    string
	ModTime int64
}

func (i *Info) ToString() string {
	return fmt.Sprintf("Action: %v; Path: %s; ModTime: %d;", i.Action, i.Path, i.ModTime)
}

const PATH = "TestDir"
