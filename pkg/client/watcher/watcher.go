package watcher

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

	p "github.com/preegnees/gobox/pkg/fileWorker/protocol"
	u "github.com/preegnees/gobox/pkg/fileWorker/utils"
)

var (
	ERROR__IOUTIL_READDIR_METHOD__ = errors.New("Err ioutil.ReadDir")
	ERROR__WATCHER_ERRORS_METHOD__ = errors.New("Err watcher.Errors")
	ERROR__NEW_WATCHER_METHOD__    = errors.New("Err fsnotify.NewWatcher")
	ERROR__OS_STAT_METHOD__        = errors.New("Err os.Stat cnf.Dir")
	ERROR__IS_NOT_DIR__            = errors.New("is not dir")
)

var _ IWatcher = (*Watcher)(nil)

// IWatcher. интерфейс для взаимодействия с пакетом
type IWatcher interface {
	Watch() error
	GetEventChan() chan p.Info
}

// ConfWatcher. Конфигурация для мониторинга
type ConfWatcher struct {
	Ctx context.Context
	Log *logrus.Logger
	Dir string
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
	EventCh chan p.Info
}

func (w *Watcher) ToString() string {

	return fmt.Sprintf(
		"context: %v, levelLog: %s, dir: %s, fsnotify.Watcher: %v, eventCh: %v",
		w.ctx, w.log.Level, w.dir, w.watcher, w.EventCh,
	)
}

// New. создает новый наблюдатель
func New(cnf ConfWatcher) (*Watcher, error) {

	cnf.Log.Debug(fmt.Sprintf("[watcher.New()] struct cnf: %v;", cnf.ToString()))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cnf.Log.Error(fmt.Errorf("[watcher.New()] New Watcher fsnotify error, err: %w;", err))
		return nil, fmt.Errorf("%w (%s);", ERROR__NEW_WATCHER_METHOD__, err.Error())
	}

	f, err := os.Stat(cnf.Dir)
	if err != nil {
		cnf.Log.Error(fmt.Errorf("[watcher.New()] stat error, err: %w, path: %s;", err, cnf.Dir))
		return nil, fmt.Errorf("%w (%s);", ERROR__OS_STAT_METHOD__, err.Error())
	}

	if !f.IsDir() {
		cnf.Log.Error(fmt.Errorf("[watcher.New()] path: %s is not dir", cnf.Dir))
		return nil, fmt.Errorf("%w;", ERROR__IS_NOT_DIR__)
	}

	ctxwrap, cancel := context.WithCancel(cnf.Ctx)

	cnf.Log.Debug("[watcher.New()] watcher creating;")

	return &Watcher{
		ctx:     ctxwrap,
		cancel:  cancel,
		watcher: watcher,
		log:     cnf.Log,
		dir:     cnf.Dir,
		EventCh: make(chan p.Info),
	}, nil
}

// Watch. Запускает мониторинг
func (w *Watcher) Watch() error {

	w.log.Debug(fmt.Sprintf("[watcher.Watch()] struct Watch: %v;", w.ToString()))

	defer w.watcher.Close()
	defer w.cancel()

	w.add(w.dir)

	if err := w.onStart(w.dir); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			default:
				time.Sleep(2 * time.Second)
				_, err := os.Stat(w.dir)
				if err != nil {
					w.log.Error(fmt.Errorf("[watcher.Watch()] MAIN DIR REMOVED;"))
					w.log.Debug(fmt.Sprintf("[watcher.Watch()] CREATE MAIN DIR;"))
					if err := os.MkdirAll(w.dir, 0777); err != nil {
						//TODO(send to user and server)
						w.log.Error(fmt.Errorf("[watcher.Watch()] CANT CREATE NEW MAIN DIR f*ck;"))
					}
				}
			}
		}
	}()

	for {
		select {
		case <-w.ctx.Done():

			w.log.Debug(fmt.Sprintf("[watcher.Watch()] context done;"))
			return nil
		case err, ok := <-w.watcher.Errors:

			if !ok {
				w.log.Error(fmt.Errorf("[watcher.Watch()] chan fsnotify errors closed;"))
				return nil
			}

			w.log.Error(fmt.Errorf("[watcher.Watch()] fsnotify error, err: %w;", err))
			return fmt.Errorf("%w (%s);", ERROR__WATCHER_ERRORS_METHOD__, err.Error())
		case event, ok := <-w.watcher.Events:

			if !ok {
				w.log.Error(fmt.Errorf("[watcher.Watch()] chan fsnotify event closed;"))
				return nil
			}

			w.log.Debug(fmt.Sprintf("[watcher.Watch()] action %d, event: %s;", event.Op, event.Name))

			pass := false
			for _, val := range u.IGNORE_STRS {
				if strings.Contains(event.Name, val) {
					w.log.Debug(fmt.Sprintf("[watcher.Watch()] name: %s include substr: %s;", event.Name, val))
					pass = true
				}
			}

			if pass {
				continue
			}

			// Тут может быть задержка, из-за возможных ошибок в u.IsFolder (см.)
			if event.Has(fsnotify.Write) {
				w.log.Debug(fmt.Sprintf("[watcher.Watch()] write to file: %s;", event.Name))

				isFolder := u.IsFolder(w.log, event.Name)

				// Это нужно, чтобы не было уведолмления о записи от вышележащих папок
				// Например: folder1/folder2/file.txt, при изменении file.txt сроботают также folder1 && 2
				if !isFolder {
					w.sendChange(event)
				}
			}

			if event.Has(fsnotify.Remove) {
				w.log.Debug(fmt.Sprintf("[watcher.Watch()] remove file: %s;", event.Name))

				w.sendChange(event)

				w.remove(event.Name)
			}

			// Тут может быть задержка, из-за возможных ошибок в u.IsFolder (см.)
			if event.Has(fsnotify.Create) {
				w.log.Debug(fmt.Sprintf("[watcher.Watch()] create file: %s;", event.Name))

				w.sendChange(event)

				isFolder := u.IsFolder(w.log, event.Name)
				if isFolder {
					w.add(event.Name)
				}
			}
		}
	}
}

// GetEventChan. Получение канала для получения событий
func (w *Watcher) GetEventChan() chan p.Info {

	w.log.Debug(fmt.Sprintf("[watcher.GetEventChan()];"))

	return w.EventCh
}

// add. добавляет путь в список наблюдаемых путей. ошибка не возвращется потому что она не важна
func (w *Watcher) add(path string) {

	w.log.Debug(fmt.Sprintf("[watcher.add()] path: %s;", path))

	if err := w.watcher.Add(path); err != nil {
		w.log.Error(fmt.Errorf("[watcher.add()] err: %w, path: %s;", err, path))
	}
}

// remove. удаляет путь из списка наблюдаемых путей. ошибка не возвращется потому что она не важна
func (w *Watcher) remove(path string) {

	w.log.Debug(fmt.Sprintf("[watcher.remove()] path: %s;", path))

	if err := w.watcher.Remove(path); err != nil {
		w.log.Error(fmt.Errorf("[watcher.remove()] err: %w, path: %s;", err, path))
	}
}

// onStart. загружает в паять метаданные файловой системы
func (w *Watcher) onStart(path string) error {

	w.log.Debug(fmt.Sprintf("[watcher.onStart()] path: %s;", path))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		w.log.Error(fmt.Errorf("[watcher.onStart()] err: %w, path: %s;", err, path))
		return fmt.Errorf("%w (%s);", ERROR__IOUTIL_READDIR_METHOD__, err.Error())
	}

	for _, v := range files {
		curPath := filepath.Join(path, v.Name())
		isFolder := u.IsFolder(w.log, curPath)

		if isFolder {
			w.add(curPath)
			w.onStart(curPath)
		}
	}

	w.log.Debug(fmt.Sprintf("[watcher] onStart() allFolders: %s", w.watcher.WatchList()))

	return nil
}

// sendChange. отправляет в канал изменения фаловой системы
func (w *Watcher) sendChange(event fsnotify.Event) {

	w.log.Debug(fmt.Sprintf("[watcher.sendChange()] action: %d, path: %s;", event.Op, event.Name))

	var modTime int64 = 0
	var hash string = ""
	var isFolder bool = false

	if !event.Op.Has(fsnotify.Remove) {
		modTime = u.GetModTime(w.log, event.Name)
		hash = u.GetHash(w.log, event.Name)
		isFolder = u.IsFolder(w.log, event.Name)
		if modTime == 0 || hash == "" {
			w.log.Error(fmt.Errorf("[watcher.sendChange()] err in getModTime or has, action: %d, path: %s;", event.Op, event.Name))
			return
		}
	} else {
		modTime = 0
		hash = ""
		isFolder = false
	}

	newEvent := p.Info{
		Action:   event.Op,
		Path:     event.Name,
		ModTime:  modTime,
		Hash:     hash,
		IsFolder: isFolder,
	}

	w.EventCh <- newEvent

	w.log.Debug(fmt.Sprintf("[watcher.sendChange()] sent info: %s;", newEvent.ToString()))
}
