package watcher

/*
	Пакет переписан в новой ветке.
	Чтобы понять был ли переименовывание,
нужно будет сравнить хеши в базе, с только что созданным, файлом или папкой!

	Не тестировались новые изменения!

	При старте программы нужно получить структуру файловой директории и сравнии ее с базой данных,
скорее всего данный пакет не будет этим заниматься.
*/

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"

	protocol "github.com/preegnees/gobox/pkg/fileWorker/protocol"
	uploader "github.com/preegnees/gobox/pkg/fileWorker/uploader"
	utils "github.com/preegnees/gobox/pkg/fileWorker/utils"
)

type IWatcher interface {
	Run() error
}

// ConfWatcher. ...
type ConfWatcher struct {
	Ctx      context.Context
	Log      *logrus.Logger
	Dir      string
	PrintErr func(desc string, arg string, err error) error
	Client   uploader.IClient
}

// DirWatcher. ...
type DirWatcher struct {
	ctx      context.Context
	cancel   context.CancelFunc
	log      *logrus.Logger
	watcher  *fsnotify.Watcher
	dir      string
	client   uploader.IClient
	printErr func(string, string, error) error
}

// New. Crete new watcher
func New(cnf ConfWatcher) (*DirWatcher, error) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	f, err := os.Stat(cnf.Dir)
	if err != nil {
		return nil, err
	}

	if !f.IsDir() {
		return nil, errors.New(fmt.Sprintf("path: %s is not dir", cnf.Dir))
	}

	var printE func(string, string, error) error
	if cnf.PrintErr == nil {
		printE = func(description string, arg string, err error) error {
			e := fmt.Errorf("[watcher] (err: %w) %s: %s", err, description, arg)
			cnf.Log.Error(e)
			return e
		}
	} else {
		printE = cnf.PrintErr
	}

	ctxwrap, cancel := context.WithCancel(cnf.Ctx)

	cnf.Log.Println("watcher creating")

	return &DirWatcher{
		ctx:      ctxwrap,
		cancel:   cancel,
		watcher:  watcher,
		log:      cnf.Log,
		dir:      cnf.Dir,
		client:   cnf.Client,
		printErr: printE,
	}, nil
}

// Run. Run watcher
func (d *DirWatcher) Run() error {

	defer d.watcher.Close()
	defer d.cancel()

	if err := d.add(d.dir); err != nil {
		return err
	}

	if err := d.onStart(d.dir); err != nil {
		return err
	}

	for {
		select {
		case <-d.ctx.Done():

			d.log.Debug("[watcher] context done")
			return nil
		case err, ok := <-d.watcher.Errors:

			if !ok {
				return d.printErr("chan fsnotify errors closed", "", nil)
			}
			d.log.Error(fmt.Errorf("[watcher] fsnotify error: %w", err))
			return err
		case event, ok := <-d.watcher.Events:

			if !ok {
				return d.printErr("chan fsnotify event closed", "", nil)
			}

			pass := false
			for _, v := range utils.IGNORE_STRS {
				if strings.Contains(event.Name, v) {
					pass = true
				}
			}

			if pass {
				d.log.Debug(fmt.Sprintf("[watcher] path include IGNORE_STRS, path: %s", event.Name))
				continue
			}

			if event.Has(fsnotify.Write) {
				d.log.Debug(fmt.Sprintf("[watcher] write to file: %s", event.Name))
				if err := d.sendChange(event); err != nil {
					return err
				}
			}

			if event.Has(fsnotify.Remove) {
				d.log.Debug(fmt.Sprintf("[watcher] remove file: %s", event.Name))
				if err := d.sendChange(event); err != nil {
					return err
				}

				isFolder, err := utils.IsFolder(d.printErr, d.log, event.Name)
				if err != nil {
					return d.printErr("", "", err)
				}
				if isFolder {
					if err = d.remove(event.Name); err != nil {
						return d.printErr("", "", err)
					}
				}
			}

			if event.Has(fsnotify.Create) {
				d.log.Debug(fmt.Sprintf("[watcher] create file: %s", event.Name))
				if err := d.sendChange(event); err != nil {
					return err
				}

				isFolder, err := utils.IsFolder(d.printErr, d.log, event.Name)
				if err != nil {
					return d.printErr("", "", err)
				}
				if isFolder {
					if err = d.add(event.Name); err != nil {
						return d.printErr("", "", err)
					}
				}
			}
		}
	}
}

// add. add path to whatcher pull for monitoring
func (d *DirWatcher) add(path string) error {

	d.log.Debug(fmt.Sprintf("[watcher] add(): %s", path))

	if err := d.watcher.Add(path); err != nil {
		return d.printErr("add(), path", path, err)
	}
	return nil
}

// remove. remove folder from watcher
func (d *DirWatcher) remove(path string) error {

	d.log.Debug(fmt.Sprintf("[watcher] remove(): %s", path))

	if err := d.watcher.Remove(path); err != nil {
		return d.printErr("[watcher] remove(), path", path, err)
	}
	return nil
}

// onStart. initing
func (d *DirWatcher) onStart(path string) error {

	d.log.Debug(fmt.Sprintf("[watcher] onStart(): %s", path))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return d.printErr("", "", err)
	}

	for _, v := range files {
		curPath := filepath.Join(path, v.Name())

		isFolder, err := utils.IsFolder(d.printErr, d.log, curPath)
		if err != nil {
			return d.printErr("", "", err)
		}

		if isFolder {
			if err := d.add(curPath); err != nil {
				return d.printErr("", "", err)
			}
			if err := d.onStart(curPath); err != nil {
				return d.printErr("", "", err)
			}
		}
	}
	d.log.Debug(fmt.Sprintf("[watcher] onStart() allFolders: %s", d.watcher.WatchList()))
	return nil
}

// sendChange. sending new change (write, remove, create)
func (d *DirWatcher) sendChange(event fsnotify.Event) error {

	modTime, err := utils.GetModTime(d.printErr, event.Name)
	if err != nil {
		return err
	}

	hash, err := utils.GetHash(d.printErr, d.log, event.Name)

	newEvent := protocol.Info{
		Action:  event.Op,
		Path:    event.Name,
		ModTime: modTime,
		Hash:    hash,
	}

	done := make(chan struct{})

	go func() {
		d.client.Send(newEvent)
		done <- struct{}{}
	}()

	select {
	case <-time.After(2 * time.Second):
		return d.printErr("sendChange() err over timeout", newEvent.Path, nil)
	case <-done:
		close(done)
		done = nil
	}

	return nil
}
