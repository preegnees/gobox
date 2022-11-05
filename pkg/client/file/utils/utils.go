package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"

	er "github.com/preegnees/gobox/pkg/client/errors"
)

var (
	IGNORE_STRS = []string{"~", "__gobox__", "tmp", "temp", "TEMP", "TMP"}
)

// GetModTime. Получение времения последней модификации
func GetModTime(log *logrus.Logger, path string) (int64, error) {

	log.Debug(fmt.Sprintf("[utils.GetModTime()] path: %s;", path))

	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf(
			"[utils.GetModTime()] (os.Stat) fileName: %s, err: %v, werr: %w;",
			path, err, er.ERROR__GET_METADATA__,
		)
	}

	modTime := fileInfo.ModTime().UTC().UnixMicro()

	log.Debug(fmt.Sprintf("[utils.GetModTime()] path: %s, modTime: %d;", path, modTime))

	return modTime, nil
}

// IsFolder. Показывает, является ли путь папкой или нет
func IsFolder(log *logrus.Logger, path string) (bool, error) {

	log.Debug(fmt.Sprintf("[utils.IsFolder()] path: %s;", path))

	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf(
			"[utils.IsFolder()] (os.Stat) fileName: %s, err: %v, werr: %w;",
			path, err, er.ERROR__GET_METADATA__,
		)
	}
	isFolder := fileInfo.IsDir()

	log.Debug(fmt.Sprintf("[utils.IsFolder()] path: %s, isFolder: %v;", path, isFolder))

	return isFolder, nil
}

// GetHash. Возвращает хеш файла или папки
func GetHash(log *logrus.Logger, path string) (string, error) {

	log.Debug(fmt.Sprintf("[utils.GetHash()] path: %s;", path))

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf(
			"[utils.GetHash()] (os.Open) fileName: %s, err: %v, werr: %w;",
			path, err, er.ERROR__GET_METADATA__,
		)
	}
	defer f.Close()

	isFolder, err := IsFolder(log, path)
	if err != nil {
		return "", err
	}

	if isFolder {
		h := sha256.New()

		_, err = h.Write([]byte(path))
		if err != nil {
			return "", fmt.Errorf(
				"[utils.GetHash()] (h.Write) fileName: %s, err: %v, werr: %w;",
				path, err, er.ERROR__GET_METADATA__,
			)
		}

		hash := hex.EncodeToString(h.Sum(nil))

		log.Debug(fmt.Sprintf("[utils.GetHash()] folder: %s, hash: %s", path, hash))

		return hash, nil
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf(
			"[utils.GetHash()] (io.Copy) fileName: %s, err: %v, werr: %w;",
			path, err, er.ERROR__GET_METADATA__,
		)
	}

	hash := hex.EncodeToString(h.Sum(nil))

	log.Debug(fmt.Sprintf("[utils.GetHash()] file: %s, hash: %s", path, hash))

	return hash, nil
}
