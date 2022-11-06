package saver

import (
	"os"
	"testing"
	"time"
)

var TEST_FILE = "TEST_FILE.txt"

func TestGetPath(t *testing.T) {
	s := saver{}
	mainPath := "hello\\world\\newFolder\\new.txt"
	want := "hello\\world\\newFolder\\" + PREFFIX + "new.txt"
	newPath := s.getPath(mainPath)
	t.Log(newPath)
	if newPath != want {
		panic("newPath != want")
	}

	oldPath := s.getPath(newPath)
	t.Log(oldPath)
	if oldPath != mainPath {
		panic("oldPath != mainPath")
	}
}

func TestResizeFile(t *testing.T) {

	f, err := os.Create(TEST_FILE)
	if err != nil {
		panic(err)
	}
	f.Close()

	defer os.Remove(TEST_FILE)

	s := saver{
		storage: make(map[string]*os.File),
	}

	file, err := os.OpenFile(TEST_FILE, os.O_RDWR, 0777)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	s.storage[TEST_FILE] = file

	startSize := int64(1024*50)
	data := make([]byte, startSize)
	file.Write(data)

	stat, err := os.Stat(TEST_FILE)
	if err != nil {
		panic(err)
	}

	if startSize != stat.Size() {
		panic("startSize != stat.Size()")
	}

	newSize1 := int64(1024*50 + 1024*100)
	if err := s.resize(TEST_FILE, newSize1); err != nil {
		panic(err)
	}

	stat, err = os.Stat(TEST_FILE)
	if err != nil {
		panic(err)
	}

	if stat.Size() != newSize1 {
		panic("stat.Size() != newSize1")
	}

	newSize2 := int64(1024*50 - 1024*30)
	if err := s.resize(TEST_FILE, newSize2); err != nil {
		panic(err)
	}

	stat, err = os.Stat(TEST_FILE)
	if err != nil {
		panic(err)
	}

	t.Log(stat.Size())

	if stat.Size() != newSize2 {
		panic("stat.Size() != newSize1")
	}
}

func TestChangeTime(t *testing.T) {

	f, err := os.Create(TEST_FILE)
	if err != nil {
		panic(err)
	}
	f.Close()

	defer os.Remove(TEST_FILE)

	s := saver{
		storage: make(map[string]*os.File),
	}

	file, err := os.OpenFile(TEST_FILE, os.O_RDWR, 0777)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	s.storage[TEST_FILE] = file

	newTime := time.Now().UnixMicro()

	if err := s.changeModTime(TEST_FILE, newTime); err != nil {
		panic(err)
	}

	stat, err := os.Stat(TEST_FILE)
	if err != nil {
		panic(err)
	}

	if newTime != stat.ModTime().UnixMicro() {
		panic("newTime != stat.ModTime().UnixMicro()")
	}
}
