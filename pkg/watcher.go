package watcher

// пакет не готов!

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Info struct {
	Action    fsnotify.Op
	Path      string
	IsDir     bool
	ModifTime time.Time
}

type DirWatcher struct {
	watcher   *fsnotify.Watcher
	log       *log.Logger
	parentDir string
	readerCh  chan Info
	errCh     chan error
	ctx       context.Context
	cancel    context.CancelFunc
}

func New(ctx context.Context, log *log.Logger, parentDir string, errCh chan error) (*DirWatcher, error) {

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(parentDir)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	return &DirWatcher{
		watcher:   w,
		log:       log,
		parentDir: parentDir,
		readerCh:  make(chan Info),
		errCh:     errCh,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

func (d *DirWatcher) Run() {

	defer d.watcher.Close()
	defer d.cancel()

	for {
		select {
		case <-d.ctx.Done():
			return
		default:
		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				d.log.Println("Write to file:", event)
			}
			if event.Has(fsnotify.Chmod) {
				d.log.Println("Change mod file:", event)
			}
			if event.Has(fsnotify.Remove) {
				log.Println("Remove file:", event)
				d.remove(event.Name)
			}
			if event.Has(fsnotify.Rename) {
				log.Println("Rename file:", event)
				d.remove(event.Name)
			}
			if event.Has(fsnotify.Create) {
				log.Println("Create file:", event)
				d.add(event.Name)
			}
		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			d.sendErr(err, true)
		}
	}
}

func (d *DirWatcher) add(path string) {

	err := d.watcher.Add(path)
	d.sendErr(err, false)
}

func (d *DirWatcher) remove(path string) {

	err := d.watcher.Remove(path)
	d.sendErr(err, false)
}

func (d *DirWatcher) isDir(path string) bool {

	fileInfo, err := os.Stat(path)
	if err != nil {
		d.sendErr(err, false)
		return false
	}

	return fileInfo.IsDir()
}

func (d *DirWatcher) onStart(path string) {

	files, err := ioutil.ReadDir(path)
	if err != nil {
		d.sendErr(err, false)
	}

	for _, v := range files {
		curPath := filepath.Join(path, v.Name())

		d.add(curPath)

		if d.isDir(curPath) {
			d.onStart(curPath)
		}
	}
}

func (d *DirWatcher) sendErr(err error, stop bool) {

	go func() {
		if stop {
			d.cancel()
		} else {
			d.errCh <- err
		}
	}()
}
