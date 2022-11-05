package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	cl "github.com/preegnees/gobox/pkg/client/client"
	pc "github.com/preegnees/gobox/pkg/client/file/protocol"
	"github.com/sirupsen/logrus"
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

	interErr := func(id int, cancel context.CancelFunc, err error) {
		t.Log(fmt.Sprintf("id: %d, err: %v", id, err))
	}

	interDev := func(info pc.Info) {
		t.Log(fmt.Sprintf("info: %s", info.ToString()))
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	cnf := ConfWatcher{
		Ctx: ctx,
		Log: logger,
		Dir: PATH,
		Client: &cli{
			intersepterErr: interErr,
			intersepterDev: interDev,
		},
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

	fileOK := false
	folderOK := false
	OK := false
	f1 := "file1.txt"
	f2 := "folder"
	fileName1 := filepath.Join(PATH, f1)
	folderName1 := filepath.Join(PATH, f2)

	interErr := func(id int, cancel context.CancelFunc, err error) {
		t.Log(fmt.Sprintf("id: %d, err: %v", id, err))
	}

	interDev := func(info pc.Info) {
		t.Log(fmt.Sprintf("info: %s", info.ToString()))
		if info.Path == fileName1 {
			fileOK = true
		}
		if info.Path == folderName1 {
			folderOK = true
		}
		if fileOK && folderOK {
			OK = true
		}
		if OK {
			cancel()
		}
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	cnf := ConfWatcher{
		Ctx: ctx,
		Log: logger,
		Dir: PATH,
		Client: &cli{
			intersepterErr: interErr,
			intersepterDev: interDev,
		},
	}

	dw, err := New(cnf)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		dw.Watch()
		wg.Done()
	}()

	chRun := make(chan struct{})

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
		wg.Done()
	}()

	timer := time.After(1 * time.Second)

	select {
	case <-timer:
		chRun <- struct{}{}
	}

	wg.Wait()
}
