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
		Log: logger,
		Dir: PATH,
		Ctx: context.TODO(),
	}

	uploader, err := New(confUploader)
	if err != nil {
		panic(err)
	}

	go func() {
		err := uploader.Upload()
		if err != nil {
			panic(err)
		}
	}()

	uploaderCh := uploader.GetEventChan()

	for {
		select {
		case _, ok := <-uploaderCh:
			if !ok {
				t.Log("chan event uploader closed")
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

	files := []map[string]bool{
		{filepath.Join(PATH, "test", "file1.exe"): true},
		{filepath.Join(PATH, "test"): false},
		{filepath.Join(PATH, "f2.html"): true},
		{filepath.Join(PATH, fmt.Sprintf("file%s", utils.IGNORE_STRS[0])): true},
	}

	if err := createFile(files); err != nil {
		panic(err)
	}

	logger := logrus.New()
	// logger.SetLevel(logrus.DebugLevel)

	countAll := 0
	countFolders := 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	confUploader := ConfUploader{
		Log: logger,
		Dir: PATH,
		Ctx: ctx,
	}

	uploader, err := New(confUploader)
	if err != nil {
		panic(err)
	}

	go func() {
		err := uploader.Upload()
		if err != nil {
			panic(err)
		}
	}()

	uploaderCh := uploader.GetEventChan()

	for {
		select {
		case e, ok := <-uploaderCh:
			if !ok {
				if countAll != 3 || countFolders != 1 {
					panic(fmt.Sprintf("countAll:%d || countFolders:%d", countAll, countFolders))
				}
				t.Log("err uploader eventch closed")
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
	files := make([]map[string]bool, 0, limit)
	for i := 0; i < limit; i++ {
		newName := filepath.Join(PATH, fmt.Sprintf("file%d", i))
		files = append(files, map[string]bool{newName: true})
	}

	if err := createFile(files); err != nil {
		panic(err)
	}

	logger := logrus.New()
	// logger.SetLevel(logrus.DebugLevel)

	count := 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	confUploader := ConfUploader{
		Log: logger,
		Dir: PATH,
		Ctx: ctx,
	}

	uploader, err := New(confUploader)
	if err != nil {
		panic(err)
	}

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

	uploaderCh := uploader.GetEventChan()

	for {
		select {
		case e, ok := <-uploaderCh:
			if !ok {
				if count >= limit || count == 0 {
					panic(fmt.Sprintf("count:%d", count))
				}
				t.Log("err uploader eventch closed")
				return
			}
			t.Log(e.ToString())
			count++
		}
	}
}

func createFile(fileNames []map[string]bool) error {
	for _, f := range fileNames {
		
		for k, v := range f {
			if v {
				if err := os.MkdirAll(filepath.Dir(k), 0770); err != nil {
					return err
				}
	
				f, err := os.Create(k)
				if err != nil {
					return err
				}
				f.Close()
			} else {
				if err := os.MkdirAll(k, 0770); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
