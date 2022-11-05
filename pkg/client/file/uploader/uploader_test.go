package uploader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	cl "github.com/preegnees/gobox/pkg/client/client"
	pc "github.com/preegnees/gobox/pkg/client/file/protocol"
	ut "github.com/preegnees/gobox/pkg/client/file/utils"
)

const PATH = "TestDir"

var _ cl.IClient = (*cli)(nil)

type cli struct {
	intersepterErr func(int, context.CancelFunc, error)
	intersepterDev func(pc.Info)
}

func (c *cli) SendError(id int, cancel context.CancelFunc, err error) {
	c.intersepterErr(id, cancel, err)
}

func (c *cli) SendDeviation(info pc.Info) {
	c.intersepterDev(info)
}

func TestUploadFilesIfDirExists(t *testing.T) {

	if err := os.MkdirAll(PATH, 0777); err != nil {
		t.Error(err)
	}

	defer os.RemoveAll(PATH)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	interErr := func(id int, cancel context.CancelFunc, err error) {
		t.Log(fmt.Sprintf("id: %d, err: %v", id, err))
	}

	interDev := func(info pc.Info) {
		t.Log(fmt.Sprintf("info: %s", info.ToString()))
	}

	confUploader := ConfUploader{
		Log: logger,
		Dir: PATH,
		Ctx: context.TODO(),
		Client: &cli{
			intersepterErr: interErr,
			intersepterDev: interDev,
		},
	}

	uploader, err := New(confUploader)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		uploader.Upload()
		wg.Done()
	}()

	wg.Wait()
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
		{filepath.Join(PATH, fmt.Sprintf("file%s", ut.IGNORE_STRS[0])): true},
	}

	if err := createFile(files); err != nil {
		panic(err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	countAll := 0
	countFolders := 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interErr := func(id int, cancel context.CancelFunc, err error) {
		t.Log(fmt.Sprintf("id: %d, err: %v", id, err))
	}

	interDev := func(info pc.Info) {
		t.Log(fmt.Sprintf("info: %s", info.ToString()))
		countAll++
		if info.IsFolder {
			countFolders++
		}
		if countAll == 3 && countFolders == 1 {
			cancel()
		}
	}

	confUploader := ConfUploader{
		Log: logger,
		Dir: PATH,
		Ctx: ctx,
		Client: &cli{
			intersepterErr: interErr,
			intersepterDev: interDev,
		},
	}

	uploader, err := New(confUploader)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		uploader.Upload()
		wg.Done()
	}()
	wg.Wait()
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
	logger.SetLevel(logrus.DebugLevel)

	count := 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interErr := func(id int, cancel context.CancelFunc, err error) {
		t.Log(fmt.Sprintf("id: %d, err: %v", id, err))
	}

	interDev := func(info pc.Info) {
		t.Log(fmt.Sprintf("info: %s", info.ToString()))
		count++
	}

	confUploader := ConfUploader{
		Log: logger,
		Dir: PATH,
		Ctx: ctx,
		Client: &cli{
			intersepterErr: interErr,
			intersepterDev: interDev,
		},
	}

	uploader, err := New(confUploader)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		uploader.Upload()
		wg.Done()
	}()

	go func() {
		<-time.After(5 * time.Millisecond)
		cancel()
	}()

	wg.Wait()
	if count >= limit {
		panic(fmt.Sprintf("count:%d", count))
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
