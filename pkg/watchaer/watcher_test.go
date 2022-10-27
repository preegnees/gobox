package watcher

import (
	"context"
	"os"
	"testing"
	"time"

	// "github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

const PATH = "TestDir"

func TestMain(t *testing.M) {

	if err := os.MkdirAll(PATH, 0777); err != nil {
		panic(err)
	}

	// defer func() {
	// 	err := os.RemoveAll(PATH)
	// 	fmt.Println("hello")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }()

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
	
	dw.Stop()

	err = os.RemoveAll(PATH)
	if err != nil {
		panic(err)
	}

	os.Exit(exitVal)
}

func TestOpenTestDir(t *testing.T) {

	f, err := os.Open(PATH)
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
}
