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

	utils "github.com/preegnees/gobox/pkg/fileWorker/utils"
)

const UPLOAD_CODE = 100

type Uploader struct {
	Ctx      context.Context
	Log      *logrus.Logger
	Dir      string
	PrintErr func(string, string, error) error
	Client   utils.IClient
}

func (u *Uploader) Upload() error {
	
	f, err := os.Stat(u.Dir)
	if err != nil {
		return err
	}

	if !f.IsDir() {
		return errors.New(fmt.Sprintf("path: %s is not dir", u.Dir))
	}
	
	printErrDefault := func(description string, arg string, err error) error {
		e := fmt.Errorf("[watcher] (err: %w) %s: %s", err, description, arg)
		u.Log.Error(e)
		return e
	}

	if u.PrintErr == nil {
		u.PrintErr = printErrDefault
	}

	if err := u.upload(u.Dir); err != nil {
		return err
	}
	return nil
}

func (u *Uploader) upload(path string) error {
	u.Log.Debug(fmt.Sprintf("[watcher] upload(): %s", path))

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return u.PrintErr("upload() err", "", err)
	}

	for _, v := range files {
		select {
		case <-u.Ctx.Done():
			return nil
		default:
			curPath := filepath.Join(path, v.Name())

			modTime, err := utils.GetModTime(u.PrintErr, curPath)
			if err != nil {
				return err
			}

			info := utils.Info{
				Action:  UPLOAD_CODE,
				Path:    strings.ReplaceAll(curPath, u.Dir, ""),
				ModTime: modTime,
			}

			if err := u.Client.Send(info); err != nil {
				return u.PrintErr("Method Send of Client failed", info.ToString(), err)
			}

			u.Log.Debug("Sent Info: ", info.ToString())

			isFolder, err := utils.IsFolder(u.PrintErr, *u.Log, curPath)
			if err != nil {
				return u.PrintErr("", "", err)
			}

			if isFolder {
				if err := u.upload(curPath); err != nil {
					return u.PrintErr("", "", err)
				}
			}
		}
	}
	return nil
}
