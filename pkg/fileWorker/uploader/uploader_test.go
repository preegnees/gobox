package uploader

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	utils "github.com/preegnees/gobox/pkg/fileWorker/utils"
)

type cli struct{
	Intersepter func(utils.Info)
}
func (c cli) Send(i utils.Info) error {
	c.Intersepter(i)
	return nil
}

func TestUploadFilesIfMainDirExists(t *testing.T) {

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	interseptor := func(i utils.Info) {
		t.Fail()
	}

	uploader := Uploader{
		Log: logger,
		Dir: utils.PATH,
		Client: cli{
			Intersepter: interseptor,
		},
		Ctx: context.TODO(),
	}

	err := uploader.Upload()
	if err != nil {
		t.Error(err)
	}
} 