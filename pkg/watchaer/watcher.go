package watcher

/*
	Пакет переписан в новой ветке. 
	Чтобы понять был ли переименовывание,
нужно будет сравнить хеши в базе, с только что созданным, файлом или папкой!

	Не тестировались новые изменения!
*/

import (
	"context"
	"errors"
	"fmt"
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

// Info. Inforamtion about event
type Info struct {
	Action fsnotify.Op
	Path   string
}

// DirWatcher. ...
type DirWatcher struct {
	ctx      context.Context
	log      *logrus.Logger
	watcher  *fsnotify.Watcher
	dir      string
	EventsCh chan Info
	ErrCh    chan error
}

// New. New watcher
func New(ctx context.Context, log *logrus.Logger, dir string) (*DirWatcher, error) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err // err of fsnotify
	}

	f, err := os.Stat(dir)
	if err != nil {
		return nil, err // is not exists or anything
	}

	if !f.IsDir() {
		return nil, errors.New(fmt.Sprintf("path: %s is not dir", dir))
	}

	log.Println("watcher creating")

	return &DirWatcher{
		ctx:      ctx,
		watcher:  watcher,
		log:      log,
		dir:      dir,
		EventsCh: make(chan Info),
		ErrCh:    make(chan error),
	}, nil
}

// Run. Async Run watcher
func (d *DirWatcher) Run() {

	go func() {
		d.start()
	}()
}

// start. start watcher
func (d *DirWatcher) start() {

	defer d.watcher.Close()

	d.add(d.dir)
	d.onStart(d.dir)

	sendErr := func(err error) {
		d.log.Error(fmt.Errorf("[watcher]. %w", err))
		d.ErrCh <- err
	}

	sendLogDebug := func(message string) {
		d.log.Debug(fmt.Sprintf("[watcher]. %s", message))
	}

	for {
		select {
		case <-d.ctx.Done():
		
			go sendErr(fmt.Errorf("context done"))
			return
		case err, ok := <-d.watcher.Errors:
			
			if !ok {
				go sendErr(fmt.Errorf("chan fsnotify errors closed"))
				return
			}
			go sendErr(fmt.Errorf("fsnotify error: %w", err))
			return
		case event, ok := <-d.watcher.Events:
		
			if !ok {
				go sendErr(errors.New("chan fsnotify closed"))
				return
			}

			pass := false
			for _, v := range IGNORE_STRS {
				if strings.Contains(event.Name, v) {
					pass = true
				}
			}

			if pass {
				sendLogDebug(fmt.Sprintf("path include IGNORE_STRS, path: %s", event.Name))
				continue
			}

			if event.Has(fsnotify.Write) {
				sendLogDebug(fmt.Sprintf("write to file: %s", event.Name))
				d.sendChange(event)
			}

			if event.Has(fsnotify.Remove) {
				sendLogDebug(fmt.Sprintf("remove file: %s", event.Name))
				d.sendChange(event)
				
				isFolder, err := d.isFolder(event.Name) 
				if err != nil {
					go sendErr(err)
				}	
				if isFolder {
					if err = d.remove(event.Name); err != nil {
						go sendErr(err)
					}
				}
			}

			if event.Has(fsnotify.Create) {
				sendLogDebug(fmt.Sprintf("create file: %s", event.Name))
				d.sendChange(event)
				
				isFolder, err := d.isFolder(event.Name) 
				if err != nil {
					go sendErr(err)
				}	
				if isFolder {
					if err = d.add(event.Name); err != nil {
						go sendErr(err)
					}
				}
			}
		}
	}
}

// add. add path to whatcher pull for monitoring
func (d *DirWatcher) add(path string) error {

	d.log.Debug("[watcher] add():", path)

	if err := d.watcher.Add(path); err != nil {
		return err
	}
	return nil
}

// remove. remove folder from watcher
func (d *DirWatcher) remove(path string) error {

	d.log.Debug("[watcher] remove():", path)

	if err := d.watcher.Remove(path); err != nil {
		return nil
	}
	return nil
}

// isFolder. check folder
func (d *DirWatcher) isFolder(path string) (bool, error) {

	d.log.Debug("[watcher] isFolder():", path)

	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), nil
}

// onStart. initing
func (d *DirWatcher) onStart(path string) error {

	d.log.Debug("[watcher] onStart():", path)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, v := range files {
		curPath := filepath.Join(path, v.Name())

		isFolder, err := d.isFolder(curPath)
		if err != nil {
			return err
		}

		if isFolder {
			if err := d.add(curPath); err != nil {
				return err
			}
			if err := d.onStart(curPath); err != nil {
				return err
			}
		}
	}
	d.log.Debug("[watcher] onStart() allFolders:", d.watcher.WatchList())
	return nil
}

func (d *DirWatcher) sendChange(event fsnotify.Event) {

	newEvent := Info{
		Action: event.Op,
		Path:   event.Name,
	}

	d.EventsCh <- newEvent
}
