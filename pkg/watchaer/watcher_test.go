package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	// "github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

const PATH = "TestDir"

var ch = make(chan Info, 0)
var ctx context.Context

func TestMain(t *testing.M) {

	os.MkdirAll(PATH, 0777)
	defer os.RemoveAll(PATH)


	var ctx, cancel = context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	dw, err := New(ctx, logger, PATH)
	if err != nil {
		panic(err)
	}

	dw.Run()

	exitVal := t.Run()

	os.Exit(exitVal)

}

func TestOpenTestDir(t *testing.T) {

	f, err := os.Open(PATH)
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
}

func TestCreateNewFile(t *testing.T) {

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case <-ctx.Done():
			wg.Done()
			return
		case val, ok := <-ch:
			if !ok {
				wg.Done()
				return
			}
			t.Log(val)
			wg.Done()
			// if val.Action.Has(fsnotify.Create) {
			// 	t.Log(val.Path)
			// 	wg.Done()
			// 	return
			// } else {
			// 	wg.Done()
			// 	t.Error(val.Action)
			// }
		}
	}()

	f, err := os.Create(filepath.Join(PATH, "newFile.txt"))
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	wg.Wait()
	// f.WriteString("test")
}
