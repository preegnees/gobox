package initializer

import (
	u "github.com/preegnees/gobox/pkg/fileWorker/uploader"
	w "github.com/preegnees/gobox/pkg/fileWorker/watcher"
)

func initing(uploader u.IUploader, watcher w.IWatcher) error {

	err := uploader.Upload()
	if err != nil {
		panic(err)
	}

	err = watcher.Watch()
	if err != nil {
		panic(err)
	}

	eventChWatcher := watcher.GetEventChan()
	eventChUploader := uploader.GetEventChan()

	for {
		select {
		case e, ok := <-eventChUploader:
			println(e)
			println(ok)
		case e, ok := <-eventChWatcher:
			println(e)
			println(ok)
		}
	}
}
