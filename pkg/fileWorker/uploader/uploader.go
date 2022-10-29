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

// IUploader. ...
type IUploader interface {
	Upload() error
}

type IClient interface {
	Send(protocol.Info) error
}

type Uploader struct {
	Ctx      context.Context
	Log      *logrus.Logger
	Dir      string
	PrintErr func(string, string, error) error
	Client   IClient
}

func (u *Uploader) Upload() error {

	f, err := os.Stat(u.Dir)
	if err != nil {
		return err
	}

	if !f.IsDir() {
		return errors.New(fmt.Sprintf("path: %s is not dir", u.Dir))
	}

	if u.PrintErr == nil {
		u.PrintErr = func(description string, arg string, err error) error {
			e := fmt.Errorf("[watcher] (err: %w) %s: %s", err, description, arg)
			u.Log.Error(e)
			return e
		}
	}

	if err := u.upload(u.Dir); err != nil {
		return err
	}
	return nil
}

func (u *Uploader) upload(path string) error {

	u.Log.Debug(fmt.Sprintf("[uploader] upload(): %s", path))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return u.PrintErr("[uploader] upload() err", "", err)
	}

	for _, file := range files {
		select {
		case <-u.Ctx.Done():
			return nil
		default:
			curPath := filepath.Join(path, file.Name())

			pass := false
			for _, val := range utils.IGNORE_STRS {
				if strings.Contains(curPath, val) {
					pass = true
				}
			}

			if pass {
				u.Log.Debug(fmt.Sprintf("[uploader] path include IGNORE_STRS, path: %s", curPath))
				continue
			}

			modTime, err := utils.GetModTime(u.PrintErr, curPath)
			if err != nil {
				return err
			}

			hash, err := utils.GetHash(u.PrintErr, u.Log, curPath)
			if err != nil {
				return err
			}

			isFolder, err := utils.IsFolder(u.PrintErr, u.Log, curPath)
			if err != nil {
				return u.PrintErr("[uploader]", "", err)
			}

			info := protocol.Info{
				Action:   protocol.UPLOAD_CODE,
				Path:     curPath,
				ModTime:  modTime,
				Hash:     hash,
				IsFolder: isFolder,
			}

			if err := u.Client.Send(info); err != nil {
				return u.PrintErr("[uploader] Method Send of Client failed", info.ToString(), err)
			}

			u.Log.Debug(fmt.Sprintf("[uploader] Sent Info: %s", info.ToString()))

			if isFolder {
				if err := u.upload(curPath); err != nil {
					return u.PrintErr("[uploader]", "", err)
				}
			}
		}
	}
	return nil
}
