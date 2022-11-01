package uploader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	protocol "github.com/preegnees/gobox/pkg/fileWorker/protocol"
	"github.com/preegnees/gobox/pkg/fileWorker/utils"
)

const PATH = "TestDir"

type cli struct {
	Intersepter func(protocol.Info)
}

func (c cli) Send(i protocol.Info) error {
	c.Intersepter(i)
	return nil
}

func TestUploadFilesIfDirExists(t *testing.T) {

	if err := os.MkdirAll(PATH, 0777); err != nil {
		t.Error(err)
	}

	defer os.RemoveAll(PATH)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	confUploader := ConfUploader{
		Log:      logger,
		Dir:      PATH,
		Ctx:      context.TODO(),
		PrintErr: nil,
	}

	uploader := New(confUploader)

	go func() {
		err := uploader.Upload()
		if err != nil {
			panic(err)
		}
	}()

	for {
		select {
		case e, ok := <-uploader.EventCh:
			if !ok {
				t.Log("chan event uploader closed")
			}
			if e.Action == 101 {
				close(uploader.EventCh)
				return
			}
		}
	}
}

func TestUploadFilesIfDirExistsAndIntoMoreAnythigFilesAndFolders(t *testing.T) {

	if err := os.MkdirAll(PATH, 0777); err != nil {
		panic(err)
	}

	defer func() {
		if err := os.RemoveAll(PATH); err != nil {
			panic("removeAll")
		}
	}()

	files := []string{
		filepath.Join(PATH, "test", "file1.exe"),
		filepath.Join(PATH, "test"),
		filepath.Join(PATH, "f2.html"),
		filepath.Join(PATH, fmt.Sprintf("file%s", utils.IGNORE_STRS[0])),
	}

	if err := createFile(files...); err != nil {
		panic(err)
	}

	logger := logrus.New()
	// logger.SetLevel(logrus.DebugLevel)

	countAll := 0
	countFolders := 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	confUploader := ConfUploader{
		Log:      logger,
		Dir:      PATH,
		Ctx:      ctx,
		PrintErr: nil,
	}

	uploader := New(confUploader)

	go func() {
		err := uploader.Upload()
		if err != nil {
			panic(err)
		}
	}()

	for {
		select {
		case e, ok := <-uploader.EventCh:
			if !ok {
				t.Log("err uploader eventch closed")
				return
			}
			if e.Action == 101 {
				if countAll != 3 || countFolders != 1 {
					panic(fmt.Sprintf("countAll:%d || countFolders:%d", countAll, countFolders))
				}
				return
			}
			t.Log(e.ToString())
			if e.IsFolder {
				countFolders++
			}
			countAll++
		}
	}
}

func TestUploadCheckDoneCtx(t *testing.T) {
	if err := os.MkdirAll(PATH, 0777); err != nil {
		panic(err)
	}

	defer func() {
		if err := os.RemoveAll(PATH); err != nil {
			panic("removeAll")
		}
	}()

	limit := 1000
	files := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		newName := filepath.Join(PATH, fmt.Sprintf("file%d", i))
		files = append(files, newName)
	}

	if err := createFile(files...); err != nil {
		panic(err)
	}

	logger := logrus.New()
	// logger.SetLevel(logrus.DebugLevel)

	count := 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	confUploader := ConfUploader{
		Log:      logger,
		Dir:      PATH,
		Ctx:      ctx,
		PrintErr: nil,
	}

	uploader := New(confUploader)

	go func() {
		err := uploader.Upload()
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		<-time.After(5 * time.Millisecond)
		cancel()
	}()

	for {
		select {
		case e, ok := <-uploader.EventCh:
			if !ok {
				t.Log("err uploader eventch closed")
				return
			}
			if e.Action == 101 {
				if count >= limit || count == 0 {
					panic(fmt.Sprintf("count:%d", count))
				}
				return
			}
			t.Log(e.ToString())
			count++
		}
	}
}

func createFile(fileNames ...string) error {
	for _, f := range fileNames {

		isFolder, err := utils.IsFolder(func(string, string, error) error { return nil }, logrus.New(), f)
		if err != nil {
			return err
		}
		if isFolder {
			if err := os.MkdirAll(f, 0770); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(f), 0770); err != nil {
				return err
			}

			f, err := os.Create(f)
			if err != nil {
				return err
			}
			f.Close()
		}
	}
	return nil
}
