package watcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"

	cl "github.com/preegnees/gobox/pkg/client/client"
	pc "github.com/preegnees/gobox/pkg/client/file/protocol"
	ut "github.com/preegnees/gobox/pkg/client/file/utils"
	er "github.com/preegnees/gobox/pkg/client/errors"
)

// Данный идентификатор привязан к данному пакету, и если произойдет ошибка то можно перезапустить сервис (пакет)
const IDENTIFIER = 1

// Проверка на соответсвие интерфейсу
var _ IWatcher = (*Watcher)(nil)

// IWatcher. интерфейс для взаимодействия с пакетом
type IWatcher interface {
	Watch()
}

// ConfWatcher. Конфигурация для мониторинга
type ConfWatcher struct {
	Ctx    context.Context
	Log    *logrus.Logger
	Dir    string
	Client cl.IClient
}

func (c *ConfWatcher) ToString() string {

	return fmt.Sprintf(
		"context: %v, levelLog: %s, dir: %s",
		c.Ctx, c.Log.Level, c.Dir,
	)
}

// DirWatcher. Структура наблюдателя
type Watcher struct {
	ctx     context.Context
	cancel  context.CancelFunc
	log     *logrus.Logger
	watcher *fsnotify.Watcher
	dir     string
	client  cl.IClient
}

func (w *Watcher) ToString() string {

	return fmt.Sprintf(
		"context: %v, levelLog: %s, dir: %s, fsnotify.Watcher: %v, client: %v",
		w.ctx, w.log.Level, w.dir, w.watcher, w.client,
	)
}

// New. создает новый наблюдатель
func New(cnf ConfWatcher) (*Watcher, error) {

	cnf.Log.Debug(fmt.Sprintf("[watcher.New()] struct cnf: %v;", cnf.ToString()))

	if cnf.Log == nil {
		return nil, fmt.Errorf("[watcher.New()] log is nil;")
	}

	if cnf.Client == nil {
		return nil, fmt.Errorf("[watcher.New()] client is nil;")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("[watcher.New()] New Watcher fsnotify error, err: %w;", err)
	}

	f, err := os.Stat(cnf.Dir)
	if err != nil {
		return nil, fmt.Errorf("[watcher.New()] stat error, err: %w, path: %s;", err, cnf.Dir)
	}

	if !f.IsDir() {
		return nil, fmt.Errorf("[watcher.New()] path: %s is not dir", cnf.Dir)
	}

	ctxwrap, cancel := context.WithCancel(cnf.Ctx)

	cnf.Log.Debug("[watcher.New()] watcher creating;")

	return &Watcher{
		ctx:     ctxwrap,
		cancel:  cancel,
		watcher: watcher,
		log:     cnf.Log,
		dir:     cnf.Dir,
		client:  cnf.Client,
	}, nil
}

// Watch. Запускает мониторинг
func (w *Watcher) Watch() {

	w.log.Debug(fmt.Sprintf("[watcher.Watch()] struct Watch: %v;", w.ToString()))

	defer w.watcher.Close()
	defer w.cancel()

	w.add(w.dir)

	if err := w.onStart(w.dir); err != nil {
		w.client.SendError(IDENTIFIER, w.cancel, err)
	}

	go func() {
		for {
			select {
			case <-w.ctx.Done():
				w.log.Debug(fmt.Sprintf("[watcher.Watch()] context done;"))
				return
			default:
				time.Sleep(2 * time.Second)
				_, err := os.Stat(w.dir)
				if err != nil {
					w.log.Error(fmt.Errorf("[watcher.Watch()] MAIN DIR REMOVED;"))
					w.log.Debug(fmt.Sprintf("[watcher.Watch()] CREATE MAIN DIR;"))
					if err := os.MkdirAll(w.dir, 0777); err != nil {
						w.client.SendError(IDENTIFIER, w.cancel, err)
					}
				}
			}
		}
	}()

	for {
		select {
		case <-w.ctx.Done():

			w.log.Debug(fmt.Sprintf("[watcher.Watch()] context done;"))
			return
		case err, ok := <-w.watcher.Errors:

			if !ok {
				w.client.SendError(
					IDENTIFIER,
					w.cancel,
					fmt.Errorf("[watcher.Watch()] chan fsnotify errors closed, werr: %w;", er.ERROR__WILL_CAUSE_A_STOP__),
				)
				return
			}

			w.client.SendError(
				IDENTIFIER,
				w.cancel,
				fmt.Errorf("[watcher.Watch()] fsnotify error, err: %w;", err),
			)
		case event, ok := <-w.watcher.Events:

			if !ok {
				w.client.SendError(
					IDENTIFIER,
					w.cancel,
					fmt.Errorf("[watcher.Watch()] chan fsnotify event closed;, werr: %w;", er.ERROR__WILL_CAUSE_A_STOP__),
				)
				return
			}

			w.log.Debug(fmt.Sprintf("[watcher.Watch()] action %d, event: %s;", event.Op, event.Name))

			pass := false
			for _, val := range ut.IGNORE_STRS {
				if strings.Contains(event.Name, val) {
					w.log.Debug(fmt.Sprintf("[watcher.Watch()] name: %s include substr: %s;", event.Name, val))
					pass = true
				}
			}

			if pass {
				continue
			}

			if event.Has(fsnotify.Write) {
				w.log.Debug(fmt.Sprintf("[watcher.Watch()] write to file: %s;", event.Name))

				isFolder, err := ut.IsFolder(w.log, event.Name)
				if err != nil {
					w.client.SendError(IDENTIFIER, w.cancel, err)
				}

				// Это нужно, чтобы не было уведолмления о записи от вышележащих папок
				// Например: folder1/folder2/file.txt, при изменении file.txt сроботают также folder1 && 2
				if !isFolder {
					if err := w.sendChange(event); err != nil {
						w.client.SendError(IDENTIFIER, w.cancel, err)
					}
				}
			}

			if event.Has(fsnotify.Remove) {
				w.log.Debug(fmt.Sprintf("[watcher.Watch()] remove file: %s;", event.Name))

				if err := w.sendChange(event); err != nil {
					w.client.SendError(IDENTIFIER, w.cancel, err)
				}

				w.remove(event.Name)
			}

			// Тут может быть задержка, из-за возможных ошибок в u.IsFolder (см.)
			if event.Has(fsnotify.Create) {
				w.log.Debug(fmt.Sprintf("[watcher.Watch()] create file: %s;", event.Name))

				if err := w.sendChange(event); err != nil {
					w.client.SendError(IDENTIFIER, w.cancel, err)
				}

				isFolder, err := ut.IsFolder(w.log, event.Name)
				if err != nil {
					w.client.SendError(IDENTIFIER, w.cancel, err)
				}
				if isFolder {
					w.add(event.Name)
				}
			}
		}
	}
}

// add. добавляет путь в список наблюдаемых путей. ошибка не возвращется потому что она не важна
func (w *Watcher) add(path string) {

	w.log.Debug(fmt.Sprintf("[watcher.add()] path: %s;", path))

	if err := w.watcher.Add(path); err != nil {
		w.client.SendError(IDENTIFIER, w.cancel, fmt.Errorf("[watcher.add()] (watcher.Add) path: %s, err: %w;", path, err))
	}
}

// remove. удаляет путь из списка наблюдаемых путей. ошибка не возвращется потому что она не важна
func (w *Watcher) remove(path string) {

	w.log.Debug(fmt.Sprintf("[watcher.remove()] path: %s;", path))

	if err := w.watcher.Remove(path); err != nil {
		w.client.SendError(IDENTIFIER, w.cancel, fmt.Errorf("[watcher.remove()] (watcher.Remove) path: %s, err: %w;", path, err))
	}
}

// onStart. загружает в паять метаданные файловой системы
func (w *Watcher) onStart(path string) error {

	w.log.Debug(fmt.Sprintf("[watcher.onStart()] path: %s;", path))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf(
			"[watcher.onStart()] (ioutil.ReadDir) path: %s, err: %v, werr: %w;",
			path, err, er.ERROR__GET_ALL_FILES_FROM_DIR__,
		)
	}

	for _, v := range files {
		curPath := filepath.Join(path, v.Name())
		isFolder, err := ut.IsFolder(w.log, curPath)
		if err != nil {
			return err
		}

		if isFolder {
			w.add(curPath)
			if err := w.onStart(curPath); err != nil {
				return err
			}
		}
	}

	w.log.Debug(fmt.Sprintf("[watcher.onStart()] allFolders: %s", w.watcher.WatchList()))

	return nil
}

// sendChange. отправляет в канал изменения фаловой системы
func (w *Watcher) sendChange(event fsnotify.Event) error {

	w.log.Debug(fmt.Sprintf("[watcher.sendChange()] action: %d, path: %s;", event.Op, event.Name))

	var modTime int64 = 0
	var hash string = ""
	var isFolder bool = false
	var err error

	if !event.Op.Has(fsnotify.Remove) {
		modTime, err = ut.GetModTime(w.log, event.Name)
		hash, err = ut.GetHash(w.log, event.Name)
		isFolder, err = ut.IsFolder(w.log, event.Name)
	} else {
		modTime = 0
		hash = ""
		isFolder = false
	}

	if err != nil {
		return err
	}

	newEvent := pc.Info{
		Action:   event.Op,
		Path:     event.Name,
		ModTime:  modTime,
		Hash:     hash,
		IsFolder: isFolder,
	}

	w.client.SendDeviation(newEvent)

	w.log.Debug(fmt.Sprintf("[watcher.sendChange()] sent info: %s;", newEvent.ToString()))

	return nil
}
