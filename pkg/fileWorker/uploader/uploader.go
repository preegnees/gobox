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

	protocol "github.com/preegnees/gobox/pkg/fileWorker/protocol"
	utils "github.com/preegnees/gobox/pkg/fileWorker/utils"
)

var _ IUploader = (*Uploader)(nil)


// IUploader. ...
type IUploader interface {
	Upload() error
}

// ConfUploader. ...
type ConfUploader struct {
	Ctx      context.Context
	Log      *logrus.Logger
	Dir      string
	PrintErr func(desc string, arg string, err error) error
}

// Uploader. ...
type Uploader struct {
	ctx      context.Context
	cancel   context.CancelFunc
	log      *logrus.Logger
	dir      string
	printErr func(string, string, error) error
	EventCh  chan protocol.Info
}

func New(cnf ConfUploader) *Uploader {

	ctx, cancel := context.WithCancel(cnf.Ctx)

	var printE func(string, string, error) error
	if cnf.PrintErr == nil {
		printE = func(description string, arg string, err error) error {
			e := fmt.Errorf("[uploader] (err: %w) %s: %s", err, description, arg)
			cnf.Log.Error(e)
			return e
		}
	} else {
		printE = cnf.PrintErr
	}

	cnf.Log.Println("uploader creating")

	return &Uploader{
		ctx:      ctx,
		cancel:   cancel,
		log:      cnf.Log,
		dir:      cnf.Dir,
		printErr: printE,
		EventCh:  make(chan protocol.Info),
	}
}

func (u *Uploader) Upload() error {

	defer u.cancel()

	f, err := os.Stat(u.dir)
	if err != nil {
		return err
	}

	if !f.IsDir() {
		return errors.New(fmt.Sprintf("path: %s is not dir", u.dir))
	}

	if u.printErr == nil {
		u.printErr = func(description string, arg string, err error) error {
			e := fmt.Errorf("[watcher] (err: %w) %s: %s", err, description, arg)
			u.log.Error(e)
			return e
		}
	}

	if err := u.upload(u.dir); err != nil {
		return err
	}
	return nil
}

func (u *Uploader) upload(path string) error {

	u.log.Debug(fmt.Sprintf("[uploader] upload(): %s", path))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return u.printErr("[uploader] upload() err", "", err)
	}

	var mainErr error

	for _, file := range files {
		select {
		case <-u.ctx.Done():
			break
		default:
			curPath := filepath.Join(path, file.Name())

			pass := false
			for _, val := range utils.IGNORE_STRS {
				if strings.Contains(curPath, val) {
					pass = true
				}
			}

			if pass {
				u.log.Debug(fmt.Sprintf("[uploader] path include IGNORE_STRS, path: %s", curPath))
				continue
			}

			modTime, err := utils.GetModTime(u.printErr, curPath)
			if err != nil {
				mainErr = err
				break
			}

			hash, err := utils.GetHash(u.printErr, u.log, curPath)
			if err != nil {
				mainErr = err
				break
			}

			isFolder, err := utils.IsFolder(u.printErr, u.log, curPath)
			if err != nil {
				mainErr = u.printErr("[uploader]", "", err)
				break
			}

			info := protocol.Info{
				Action:   protocol.UPLOAD_CODE,
				Path:     curPath,
				ModTime:  modTime,
				Hash:     hash,
				IsFolder: isFolder,
			}

			u.EventCh <- info

			u.log.Debug(fmt.Sprintf("[uploader] Sent Info: %s", info.ToString()))

			if isFolder {
				if err := u.upload(curPath); err != nil {
					mainErr = u.printErr("[uploader]", "", err)
					break
				}
			}
		}
	}

	u.EventCh <- protocol.Info{
		Action: (101),
	}

	if mainErr != nil {
		return mainErr
	}

	return nil
}
