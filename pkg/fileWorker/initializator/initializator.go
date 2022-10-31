package initializator

import (
	"context"

	"github.com/sirupsen/logrus"

	p "github.com/preegnees/gobox/pkg/fileWorker/protocol"
	u "github.com/preegnees/gobox/pkg/fileWorker/uploader" // перенести клиент в другое место
	w "github.com/preegnees/gobox/pkg/fileWorker/watcher"
)

var _ IInitializator = (*Initializator)(nil)

type IInitializator interface {
	Run(Initializator) error
	Stop() error
}

type Initializator struct {
	Ctx          context.Context
	Log          *logrus.Logger
	Client       u.IClient
	Uploader     u.IUploader
	Watcher      w.IWatcher
	FinalStorage []p.Info
	Dir          string
	EventCh      chan p.Info
}

func (i *Initializator) Run(cnf Initializator) error {

	ctx, cancel := context.WithCancel(cnf.Ctx)
	defer cancel()

	cnfWatcher := w.ConfWatcher{
		Ctx:      ctx,
		Log:      cnf.Log,
		Dir:      cnf.Dir,
		PrintErr: nil,
	}
	watcher, err := w.New(cnfWatcher)
	if err != nil {
		panic(err)
	}

	if err := watcher.Run(); err != nil {
		panic(err)
	}

	uploader := u.Uploader{
		Ctx:      ctx,
		Log:      cnf.Log,
		Dir:      cnf.Dir,
		PrintErr: nil,
		Client:   cnf.Client,
	}

	if err := uploader.Upload(); err != nil {
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case :
		}
	}

	return nil
}

func (i *Initializator) Stop() error {

	return nil
}
