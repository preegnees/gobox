package watcher

// пакет не готов!

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

var (
	IGNORE_STRS = []string{"~", "__gobox__"}
)

type Info struct {
	Action   fsnotify.Op
	Path     string
}

type DirWatcher struct {
	watcher   *fsnotify.Watcher
	log       *logrus.Logger
	parentDir string
	ReaderCh  chan Info
	ErrCh     chan error
	ctx       context.Context
}

func New(ctx context.Context, log *logrus.Logger, parentDir string) (*DirWatcher, error) {

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(parentDir)
	if err != nil {
		return nil, err
	}

	log.Println("watcher initialized")

	return &DirWatcher{
		watcher:   w,
		log:       log,
		parentDir: parentDir,
		ReaderCh:  make(chan Info),
		ErrCh:     make(chan error),
		ctx:       ctx,
	}, nil
}

func (d *DirWatcher) Run() {

	go func() {
		d.start()
	}()
}

func (d *DirWatcher) start() {

	defer d.watcher.Close()

	d.add(d.parentDir)
	d.onStart(d.parentDir)

	for {
		select {
		case <-d.ctx.Done():

			return
		case event, ok := <-d.watcher.Events:

			if !ok {
				return
			}

			pass := false
			for _, v := range IGNORE_STRS {
				if strings.Contains(event.Name, v) {
					pass = true
				}
			}

			if pass {
				continue
			}

			if event.Has(fsnotify.Write) {
				d.log.Debug("write:", event.Name)
				d.sendChange(event)
			}

			if event.Has(fsnotify.Remove) {
				d.log.Debug("remove:", event.Name)
				d.sendChange(event)
				if d.isFolder(event.Name) {
					d.remove(event.Name)
				}
			}

			if event.Has(fsnotify.Rename) {
				d.log.Debug("rename:", event.Name)
				d.sendChange(event)
				if d.isFolder(event.Name) {
					d.remove(event.Name)
				}
			}

			if event.Has(fsnotify.Create) {
				d.log.Debug("create:", event.Name)
				d.sendChange(event)
				if d.isFolder(event.Name) {
					d.add(event.Name)
				}
			}

		case err, ok := <-d.watcher.Errors:

			if !ok {
				return
			}
			d.sendErr(err)
		}
	}
}

func (d *DirWatcher) add(path string) {

	d.log.Debug("add():", path)

	err := d.watcher.Add(path)
	d.sendErr(err)
}

func (d *DirWatcher) remove(path string) {

	d.log.Debug("remove():", path)

	err := d.watcher.Remove(path)
	d.sendErr(err)
}

func (d *DirWatcher) isFolder(path string) bool {

	d.log.Debug("isFolder():", path)

	fileInfo, err := os.Stat(path)
	if d.sendErr(err) {
		return false
	}

	return fileInfo.IsDir()
}

func (d *DirWatcher) onStart(path string) {

	d.log.Debug("onStart():", path)

	files, err := ioutil.ReadDir(path)
	d.sendErr(err)

	for _, v := range files {
		curPath := filepath.Join(path, v.Name())
		if d.isFolder(curPath) {
			d.add(curPath)
			d.onStart(curPath)
		}
	}
	d.log.Debug("onStart() allFolders:", d.watcher.WatchList())
}

func (d *DirWatcher) sendErr(err error) bool {

	if err != nil {
		d.log.Println("sendErr():", err)
		go func() {
			d.ErrCh <- err
		}()
		return true
	}
	return false
}

func (d *DirWatcher) sendChange(event fsnotify.Event) {

	newEvent := Info{
		Action:   event.Op,
		Path:     event.Name,
	}
	d.ReaderCh <- newEvent
}
