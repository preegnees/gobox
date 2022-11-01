package initializer

import (
	"context"
	"sync"

	p "github.com/preegnees/gobox/pkg/fileWorker/protocol"
	u "github.com/preegnees/gobox/pkg/fileWorker/uploader"
	w "github.com/preegnees/gobox/pkg/fileWorker/watcher"
	"github.com/sirupsen/logrus"
)

// TODO(там есть код 101, нужно вынести его в протокол или ютилс)

type IClient interface {
	Send(p.Info) error
}

//

var _ IInitializer = (*initializer)(nil)

type IInitializer interface {
	Initize() error
}

type ConfInitializer struct {
	Ctx      context.Context
	Log      *logrus.Logger
	Uploader u.IUploader
	Watcher  w.IWatcher
	Client   IClient
}

type initializer struct {
	cnf ConfInitializer
	infos map[string]p.Info
}

func New(cnf ConfInitializer) (initializer, error) {
	return initializer{
		infos: make(map[string]p.Info),
		cnf: cnf,
	}, nil
}

func (i *initializer) Initize() error {

	go func() {
		err := i.cnf.Uploader.Upload()
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		err := i.cnf.Watcher.Watch()
		if err != nil {
			panic(err)
		}
	}()

	eventChWatcher := i.cnf.Watcher.GetEventChan()
	eventChUploader := i.cnf.Uploader.GetEventChan()

	var mx sync.Mutex
	add := func(info p.Info) {
		if eventChUploader == nil {
			return
		}
		mx.Lock()
		i.infos[info.Path] = info
		mx.Unlock()
	}

	for {
		select {
		case <-i.cnf.Ctx.Done():
			return nil
		case e, ok := <-eventChUploader:
			if !ok {
				return nil
			}
			if e.Action == 101 {
				for _, info := range i.infos {
					if err := i.cnf.Client.Send(info); err != nil {
						panic(err)
					}
				}
				if err := i.cnf.Client.Send(e); err != nil {
					panic(err)
				}
				close(eventChUploader)
				eventChUploader = nil
			}
			add(e)
		case e, ok := <-eventChWatcher:
			if !ok {
				return nil
			}
			add(e)
			if err := i.cnf.Client.Send(e); err != nil {
				panic(err)
			}
		}
	}
}
