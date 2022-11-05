package uploader

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	cl "github.com/preegnees/gobox/pkg/client/client"
	pc "github.com/preegnees/gobox/pkg/client/file/protocol"
	ut "github.com/preegnees/gobox/pkg/client/file/utils"
	er "github.com/preegnees/gobox/pkg/client/errors"
)

var IDENTIFIER = 2

var _ IUploader = (*Uploader)(nil)

// IUploader. интерфейс для взаимодействия с пакетом
type IUploader interface {
	Upload()
}

// ConfUploader. конфигурация для загрузчика
type ConfUploader struct {
	Ctx    context.Context
	Log    *logrus.Logger
	Dir    string
	Client cl.IClient
}

func (c *ConfUploader) ToString() string {

	return fmt.Sprintf(
		"context: %v, levelLog: %s, dir: %s",
		c.Ctx, c.Log.Level, c.Dir,
	)
}

// Uploader. структура загрузчика
type Uploader struct {
	ctx    context.Context
	cancel context.CancelFunc
	log    *logrus.Logger
	dir    string
	client cl.IClient
}

func (u *Uploader) ToString() string {

	return fmt.Sprintf(
		"context: %v, levelLog: %s, dir: %s, client: %v",
		u.ctx, u.log.Level, u.dir, u.client,
	)
}

// New. создает новый загрзчик
func New(cnf ConfUploader) (*Uploader, error) {

	cnf.Log.Debug(fmt.Sprintf("[uploader.New()] struct cnf: %v;", cnf.ToString()))

	if cnf.Log == nil {
		return nil, fmt.Errorf("[watcher.New()] log is nil;")
	}

	if cnf.Client == nil {
		return nil, fmt.Errorf("[watcher.New()] client is nil;")
	}

	f, err := os.Stat(cnf.Dir)
	if err != nil {
		return nil, fmt.Errorf("[uploader.New()] stat error, err: %w, path: %s;", err, cnf.Dir)
	}

	if !f.IsDir() {
		return nil, fmt.Errorf("[uploader.New()] path: %s is not dir", cnf.Dir)
	}

	ctx, cancel := context.WithCancel(cnf.Ctx)

	cnf.Log.Debug("[uploader.New()] uploader creating;")

	return &Uploader{
		ctx:    ctx,
		cancel: cancel,
		log:    cnf.Log,
		dir:    cnf.Dir,
		client: cnf.Client,
	}, nil
}

func (u *Uploader) Upload() {

	defer u.cancel()

	if err := u.upload(u.dir); err != nil {
		u.client.SendError(IDENTIFIER, u.cancel, err)
		return
	}
}

func (u *Uploader) upload(path string) error {

	u.log.Debug(fmt.Sprintf("[uploader.upload()] path: %s;", path))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf(
			"[uploader.upload()] (ioutil.ReadDir) path: %s, err: %v, werr: %w;",
			path, err, er.ERROR__GET_ALL_FILES_FROM_DIR__,
		)
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

			modTime, err := ut.GetModTime(u.log, curPath)
			hash, err := ut.GetHash(u.log, curPath)
			isFolder, err := ut.IsFolder(u.log, curPath)
			if err != nil {
				u.client.SendError(IDENTIFIER, u.cancel, err)
			}

			info := pc.Info{
				Action:   pc.UPLOAD_CODE,
				Path:     curPath,
				ModTime:  modTime,
				Hash:     hash,
				IsFolder: isFolder,
			}

			u.client.SendDeviation(info)

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
