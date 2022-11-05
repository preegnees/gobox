package saver

/*
	тут нужно как то хронить открытые файлы.
	иметь функцию открытия и закрытия файла.
	иметь функцию записи.
	перед записью нужно переименовать __gobox__
	после записи нужно поставить нужную дату и переименовать обратно

	добавить еще функцию которая меняет размер файла типа увеличивает его или уменьшает
*/

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	pc "github.com/preegnees/gobox/pkg/client/file/protocol"
	"github.com/sirupsen/logrus"
)

const PREFFIX = "__gobox__"

var _ ISaver = (*saver)(nil)

type ISaver interface {
	Open(string) error
	Close(string) error
	Write(pc.Info) error
	CreateFolder(string) error
}

type ConfSaver struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	Log    *logrus.Logger
}

type saver struct {
	ctx     context.Context
	cancel  context.CancelFunc
	log     *logrus.Logger
	storage map[string]*os.File
}

func New(cnf ConfSaver) *saver {

	return &saver{
		ctx:     cnf.Ctx,
		cancel:  cnf.Cancel,
		log:     cnf.Log,
		storage: make(map[string]*os.File),
	}
}

func (s *saver) CreateFolder(path string) error {
	return nil
}

func (s *saver) Open(path string) error {

	path, err := s.rename(path)

	f, err := os.OpenFile(path, os.O_RDWR, 0777)
	if err != nil {
		return err
	}

	oldf, ok := s.storage[path]
	if ok {
		oldf.Close()
	}

	s.storage[path] = f

	return nil
}

func (s *saver) Close(path string) error {
	return nil
}

func (s *saver) Write(info pc.Info) error {
	return nil
}

func (s *saver) changeModTime(path string, modTime int) error {
	return nil
}

func (s *saver) resize(path string, newSize int64) error {

	stat, err := os.Stat(path)
	if err != nil {
		return nil
	}

	size := stat.Size()
	if size == newSize {
		return nil
	}
	if newSize > size {
		diff := newSize - size
		data := make([]byte, diff)
		for i := range data {
			data[i] = byte(1)
		}
		file, ok := s.storage[path]
		if !ok {
			return errors.New("")
		}
		_, err := file.Write(data)
		if err != nil {
			return err
		}
		return nil
	}
	if size > newSize {
		_, ok := s.storage[path]
		if !ok {
			return errors.New("")
		}
		if err := os.Truncate(path, newSize); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (s *saver) rename(path string) (string, error) {

	newPath := s.getPath(path)
	if err := os.Rename(path, newPath); err != nil {
		return "", err
	}
	return newPath, nil
}

func (s *saver) getPath(path string) string {

	dir, file := filepath.Split(path)
	if strings.Contains(file, PREFFIX) {
		newFile := strings.ReplaceAll(file, PREFFIX, "")
		newPath := filepath.Join(dir, newFile)
		return newPath
	}
	newFile := PREFFIX + file
	newPath := filepath.Join(dir, newFile)
	return newPath
}
