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
	// logger.SetLevel(logrus.DebugLevel)

	uploader := Uploader{
		Log: logger,
		Dir: PATH,
		Client: cli{
			Intersepter: func(i protocol.Info) {
				panic("interseptet is started")
			},
		},
		Ctx: context.TODO(),
	}

	err := uploader.Upload()
	if err != nil {
		panic(err)
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

	uploader := Uploader{
		Log: logger,
		Dir: PATH,
		Client: cli{
			Intersepter: func(i protocol.Info) {
				// t.Log(i.Path)
				for _, f := range files {
					if f == i.Path {
						countAll++
						if i.IsFolder {
							countFolders++
						}
					}
				}
			},
		},
		Ctx: ctx,
	}

	err := uploader.Upload()
	if err != nil {
		panic(err)
	}

	if countAll != 3 || countFolders != 1 {
		panic(fmt.Sprintf("countAll:%d || countFolders:%d", countAll, countFolders))
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

	uploader := Uploader{
		Log: logger,
		Dir: PATH,
		Client: cli{
			Intersepter: func(i protocol.Info) {
				// t.Log(i.Path)
				for _, f := range files {
					if f == i.Path {
						count++
					}
				}
			},
		},
		Ctx: ctx,
	}

	done := make(chan struct{})
	go func() {
		<-time.After(50 * time.Millisecond)
		cancel()
		done<- struct{}{}
	}()

	err := uploader.Upload()
	if err != nil {
		panic(err)
	}

	<- done
	// t.Log("count: ", count)
	if count >= limit || count == 0 {
		panic(fmt.Sprintf("count:%d", count))
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
