package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Если эти строки будут в названии файл, то это файл будет игнорироваться
var (
	IGNORE_STRS = []string{"~", "__gobox__", "tmp", "temp", "TEMP", "TMP"}
)

var (
	ERROR_OPEN_FILE__GET_HASH_FUNC__EXCEEDED_MAX_ATTEMPT = errors.New("Err open file, maybe 'used by another process', exceeded max attempt")
	ERROR_STAT__IS_FOLDER_FUNC__EXCEEDED_MAX_ATTEMPT     = errors.New("Err of stat file, exceeded max attempt")
	ERROR_STAT__GET_MOD_TIME_FUNC__EXCEEDED_MAX_ATTEMPT  = errors.New("Err of stat file, exceeded max attempt")
	ERROR__HASH_WRITE_METHOD__                           = errors.New("Err write hash folder")
	ERROR__HASH_IO_COPY_METHOD__                         = errors.New("Err io copy hash")
)

const (
	TIMEOUT     = 1
	MAX_ATTEMPT = 10
)

// GetModTime. Получение времения последней модификации
func GetModTime(log *logrus.Logger, path string) (int64, error) {

	log.Debug(fmt.Sprintf("[utils.GetModTime()] path: %s;", path))

	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Error(fmt.Errorf("[utils.GetModTime()] err: %w, fileName: %s, timeout: %d, maxAttempt: %d;", err, path, TIMEOUT, MAX_ATTEMPT))
		for i := 1; i <= MAX_ATTEMPT; i++ {
			time.Sleep(TIMEOUT * time.Second)
			log.Debug(fmt.Errorf("[utils.GetModTime()] err: %w, path: %s, timeoutCount: %d;", err, path, i))
			fileInfo, err = os.Stat(path)
			if err == nil {
				break
			}
			if i == MAX_ATTEMPT {
				return 0, fmt.Errorf("%w (%s);", ERROR_STAT__GET_MOD_TIME_FUNC__EXCEEDED_MAX_ATTEMPT, err.Error())
			}
		}
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
		log.Error(fmt.Errorf("[utils.IsFolder()] err: %w, fileName: %s, timeout: %d, maxAttempt: %d;", err, path, TIMEOUT, MAX_ATTEMPT))
		for i := 1; i <= MAX_ATTEMPT; i++ {
			time.Sleep(TIMEOUT * time.Second)
			log.Debug(fmt.Errorf("[utils.IsFolder()] err: %w, path: %s, timeoutCount: %d;", err, path, i))
			fileInfo, err = os.Stat(path)
			if err == nil {
				break
			}
			if i == MAX_ATTEMPT {
				return false, fmt.Errorf("%w (%s);", ERROR_STAT__IS_FOLDER_FUNC__EXCEEDED_MAX_ATTEMPT, err.Error())
			}
		}
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
		log.Error(fmt.Errorf("[utils.GetHash()] err: %w, fileName: %s, timeout: %d, maxAttempt: %d;", err, path, TIMEOUT, MAX_ATTEMPT))
		for i := 1; i <= MAX_ATTEMPT; i++ {
			time.Sleep(TIMEOUT * time.Second)
			log.Debug(fmt.Errorf("[utils.GetHash()] err: %w, path: %s, timeoutCount: %d;", err, path, i))
			f, err = os.Open(path)
			if err == nil {
				break
			} else {
				f.Close()
			}
			if i == MAX_ATTEMPT {
				return "", fmt.Errorf("[utils.GetHash()] %w (%s);", ERROR_OPEN_FILE__GET_HASH_FUNC__EXCEEDED_MAX_ATTEMPT, err.Error())
			}
		}
	}
	defer f.Close()

	isFolder, err := IsFolder(log, path)
	if err != nil {
		return "", err
	}

	if isFolder {
		h := sha256.New()
		mt, err := GetModTime(log, path)
		if err != nil {
			return "", err
		}

		_, err = h.Write([]byte(strconv.Itoa(int(mt))))
		if err != nil {
			log.Error(fmt.Errorf("[utils.GetHash()] err: %w, mt: %d;", err, mt))
			return "", fmt.Errorf("%w (%s);", ERROR__HASH_WRITE_METHOD__, err.Error())
		}

		hash := hex.EncodeToString(h.Sum(nil))

		log.Debug(fmt.Sprintf("[utils.GetHash()] folder: %s, hash: %s", path, hash))

		return hash, nil
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Error(fmt.Errorf("[utils.writeForHashFolder()] err: %w;", err))
		return "", fmt.Errorf("%w (%s);", ERROR__HASH_IO_COPY_METHOD__, err.Error())
	}

	hash := hex.EncodeToString(h.Sum(nil))

	log.Debug(fmt.Sprintf("[utils.GetHash()] file: %s, hash: %s", path, hash))

	return hash, nil
}
