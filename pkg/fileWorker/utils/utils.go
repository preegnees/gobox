package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

var (
	IGNORE_STRS = []string{"~", "__gobox__"}
)

func GetModTime(printErr func(string, string, error) error, path string) (int64, error) {

	f, err := os.Stat(path)
	if err != nil {
		return -1, printErr("getModTime() err os Stat, event", path, err)
	}

	return f.ModTime().UTC().UnixMicro(), nil
}

func IsFolder(printErr func(string, string, error) error, log *logrus.Logger, path string) (bool, error) {

	log.Debug(fmt.Sprintf("[watcher] isFolder(): %s", path))

	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, printErr("isFolder(), path", path, err)
	}

	return fileInfo.IsDir(), nil
}

func GetHash(printErr func(string, string, error) error, log *logrus.Logger, fileName string) (string, error) {

	f, err := os.Open(fileName)
	if err != nil {
		return "", printErr("GetHash() err open file", fileName, err)
	}
	defer f.Close()

	isFolder, err := IsFolder(printErr, log, fileName) 
	if err != nil {
		return "", err
	}

	if isFolder {
		h := sha256.New()
		tm, err := GetModTime(printErr, fileName)
		if err != nil {
			return "", err
		}
		
		_, err = h.Write([]byte(strconv.Itoa(int(tm))))
		if err != nil {
			return "", err
		}

		hash := hex.EncodeToString(h.Sum(nil))

		log.Debug(fmt.Sprintf("GetHash() folder: %s, hash: %s", fileName, hash))
		return hash, nil
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", printErr("GetHash() err io copy file", fileName, err)
	}

	hash := hex.EncodeToString(h.Sum(nil))

	log.Debug(fmt.Sprintf("GetHash() filename: %s, hash: %s", fileName, hash))
	return hash, nil
}