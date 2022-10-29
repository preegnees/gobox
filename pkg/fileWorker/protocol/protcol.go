package protocol

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

// UPLOAD_CODE. Код, который соответсвует о том, что информация будет о файле или папки
const UPLOAD_CODE = 100

// Info. Информация, которая отправляется на сервер при просмотре файловой директории
type Info struct {
	Action   fsnotify.Op
	Path     string
	ModTime  int64
	Hash     string
	IsFolder bool
}

// ToString. Info struct в строку
func (i *Info) ToString() string {
	return fmt.Sprintf(
		"Action: %d; Path: %s; ModTime: %d; Hash: %s; IsFolder: %v;",
		i.Action, i.Path, i.ModTime, i.Hash, i.IsFolder,
	)
}
