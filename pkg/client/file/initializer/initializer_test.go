package initializer

import (
	"context"
	"testing"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	p "github.com/preegnees/gobox/pkg/client/file/protocol"
	u "github.com/preegnees/gobox/pkg/client/file/uploader"
	w "github.com/preegnees/gobox/pkg/client/file/watcher"
)

type cli struct {
	intersepter func(p.Info) 
}

func (c *cli) Send(i p.Info) error {
	c.intersepter(i)
	return nil
}

func TestInitize(t *testing.T) {
	
	PATH := "TEST_DIR"

	
	if err := os.MkdirAll(PATH, 0777); err != nil {
		panic(err)
	}
	defer func() {
		err := os.RemoveAll(PATH)
		if err != nil {
			panic(err)
		}
	}()

	f1, err := os.Create(filepath.Join(PATH, "file1.txt"))
	if err != nil {
		panic(err)
	}
	f1.Close()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	log := logrus.New()
	// log.SetLevel(logrus.DebugLevel)

	cnfUploader := u.ConfUploader{
		Ctx: ctx,
		Log: log,
		Dir: PATH,
	}
	uploader, err := u.New(cnfUploader)
	if err != nil {
		panic(err)
	}

	cnfWatchar := w.ConfWatcher{
		Ctx: ctx,
		Log: log,
		Dir: PATH,
	}
	watcher, err := w.New(cnfWatchar)
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})
	client := cli{
		intersepter: func(i p.Info) {
			t.Log(i.ToString())
			if i.Action == 101 {
				done <- struct{}{}
			}
		},
	}

	cnf := ConfInitializer{
		Ctx: ctx,
		Log: log,
		Uploader: uploader,
		Watcher: watcher,
		Client: &client,
	}

	initializer, err := New(cnf)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := initializer.Initize(); err != nil {
			panic(err)
		}
	}()

	for {
		select {
		case <-done:
			t.Log("конец")
			cancel()
			return
		}
	}
}