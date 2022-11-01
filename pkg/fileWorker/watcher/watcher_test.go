package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	// "github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

const PATH = "TestDir"

func TestOpenTestDir(t *testing.T) {

	if err := os.MkdirAll(PATH, 0777); err != nil {
		panic(err)
	}
	defer func() {
		err := os.RemoveAll(PATH)
		if err != nil {
			panic(err)
		}
	}()

	var ctx, cancel = context.WithCancel(context.TODO())
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	cnf := ConfWatcher{
		Ctx:      ctx,
		Log:      logger,
		Dir:      PATH,
		PrintErr: nil,
	}

	dw, err := New(cnf)
	if err != nil {
		panic(err)
	}

	go dw.Watch()

	f, err := os.Open(PATH)
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
}

func TestCreateFile(t *testing.T) {

	if err := os.MkdirAll(PATH, 0777); err != nil {
		panic(err)
	}
	defer func() {
		err := os.RemoveAll(PATH)
		if err != nil {
			panic(err)
		}
	}()

	var ctx, cancel = context.WithCancel(context.TODO())
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	cnf := ConfWatcher{
		Ctx:      ctx,
		Log:      logger,
		Dir:      PATH,
		PrintErr: nil,
	}

	dw, err := New(cnf)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := dw.Watch(); err != nil {
			panic(err)
		}
	}()

	chRun := make(chan struct{})
	fileName1 := filepath.Join(PATH, "file1.txt")
	folderName1 := filepath.Join(PATH, "folder")

	go func() {
		<-chRun
		f1, err := os.Create(fileName1)
		if err != nil {
			panic(err)
		}
		f1.Close()

		time.Sleep(1 * time.Second)

		err = os.MkdirAll(folderName1, 0777)
		if err != nil {
			panic(err)
		}
	}()

	timer := time.After(1 * time.Second)
	fileOK := false
	folderOK := false

	for {
		select {
		case <-timer:
			chRun <- struct{}{}
		case v, ok := <-dw.EventCh:
			t.Log("event:", v.Path)
			if !ok {
				return
			}

			if v.Path == fileName1 {
				fileOK = true
			}

			if v.Path == folderName1 {
				folderOK = true
			}

			if folderOK && fileOK {
				return
			}
		}
	}
}
