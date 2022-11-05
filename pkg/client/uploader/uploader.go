package uploader

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	p "github.com/preegnees/gobox/pkg/fileWorker/protocol"
	ut "github.com/preegnees/gobox/pkg/fileWorker/utils"
)

var (
	ERROR__IOUTIL_READDIR_METHOD__ = errors.New("Err ioutil.ReadDir")
	ERROR__OS_STAT_METHOD__        = errors.New("Err os.Stat cnf.Dir")
	ERROR__IS_NOT_DIR__            = errors.New("is not dir")
)

var _ IUploader = (*Uploader)(nil)

// IUploader. интерфейс для взаимодействия с пакетом
type IUploader interface {
	Upload() error
	GetEventChan() chan p.Info
}

// ConfUploader. конфигурация для загрузчика
type ConfUploader struct {
	Ctx context.Context
	Log *logrus.Logger
	Dir string
}

func (c *ConfUploader) ToString() string {

	return fmt.Sprintf(
		"context: %v, levelLog: %s, dir: %s",
		c.Ctx, c.Log.Level, c.Dir,
	)
}

// Uploader. структура загрузчика
type Uploader struct {
	ctx     context.Context
	cancel  context.CancelFunc
	log     *logrus.Logger
	dir     string
	eventCh chan p.Info
}

func (u *Uploader) ToString() string {

	return fmt.Sprintf(
		"context: %v, levelLog: %s, dir: %s, eventCh: %v",
		u.ctx, u.log.Level, u.dir, u.eventCh,
	)
}

// New. создает новый загрзчик
func New(cnf ConfUploader) (*Uploader, error) {

	cnf.Log.Debug(fmt.Sprintf("[uploader.New()] struct cnf: %v;", cnf.ToString()))

	f, err := os.Stat(cnf.Dir)
	if err != nil {
		cnf.Log.Error(fmt.Errorf("[uploader.New()] stat error, err: %w, path: %s;", err, cnf.Dir))
		return nil, fmt.Errorf("%w (%s);", ERROR__OS_STAT_METHOD__, err.Error())
	}

	if !f.IsDir() {
		cnf.Log.Error(fmt.Errorf("[uploader.New()] path: %s is not dir", cnf.Dir))
		return nil, fmt.Errorf("%w;", ERROR__IS_NOT_DIR__)
	}

	ctx, cancel := context.WithCancel(cnf.Ctx)

	cnf.Log.Debug("[uploader.New()] uploader creating;")

	return &Uploader{
		ctx:     ctx,
		cancel:  cancel,
		log:     cnf.Log,
		dir:     cnf.Dir,
		eventCh: make(chan p.Info),
	}, nil
}

func (u *Uploader) Upload() error {

	defer u.cancel()

	if err := u.upload(u.dir); err != nil {
		return err
	}

	close(u.eventCh)
	u.eventCh = nil

	return nil
}

func (u *Uploader) upload(path string) error {

	u.log.Debug(fmt.Sprintf("[uploader.upload()] path: %s;", path))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		u.log.Error(fmt.Errorf("[uploader.onStart()] err: %w, path: %s;", err, path))
		return fmt.Errorf("%w (%s);", ERROR__IOUTIL_READDIR_METHOD__, err.Error())
	}

	var errr error

	for _, file := range files {
		select {
		case <-u.ctx.Done():

			u.log.Debug(fmt.Sprintf("[uploader.upload()] context done;"))
			break
		default:
			curPath := filepath.Join(path, file.Name())

			u.log.Debug(fmt.Sprintf("[uploader.upload()] current path: %s;", curPath))

			pass := false
			for _, val := range ut.IGNORE_STRS {
				if strings.Contains(curPath, val) {
					u.log.Debug(fmt.Sprintf("[uploader.upload()] name: %s include substr: %s;", curPath, val))
					pass = true
				}
			}

			if pass {
				continue
			}

			modTime := ut.GetModTime(u.log, curPath)
			hash := ut.GetHash(u.log, curPath)
			isFolder := ut.IsFolder(u.log, curPath)
			if modTime == 0 || hash == "" {
				u.log.Error(fmt.Errorf("[uploader.upload()] err modTime or Hash (will not send), path: %s", curPath))
				continue
			}

			info := p.Info{
				Action:   p.UPLOAD_CODE,
				Path:     curPath,
				ModTime:  modTime,
				Hash:     hash,
				IsFolder: isFolder,
			}

			u.eventCh <- info

			u.log.Debug(fmt.Sprintf("[uploader.upload()] Sent Info: %s", info.ToString()))

			if isFolder {
				if err := u.upload(curPath); err != nil {
					errr = err
					break
				}
			}
		}
	}

	if errr != nil {
		return errr
	}

	return nil
}

// GetEventChan. Получение канала для получения событий
func (u *Uploader) GetEventChan() chan p.Info {

	u.log.Debug(fmt.Sprintf("[uploader.GetEventChan()];"))

	return u.eventCh
}
