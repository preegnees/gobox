package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

// TODO(добавить какой нибудь клиент для отсылания ошибки)

// Если эти строки будут в названии файл, то это файл будет игнорироваться
var (
	IGNORE_STRS = []string{"~", "__gobox__" /*"tmp",*/, "temp", "TEMP", "TMP"}
)

// TODO(эти ошибки больше не нужны)
var (
	ERROR_OPEN_FILE__GET_HASH_FUNC__EXCEEDED_MAX_ATTEMPT = errors.New("Err open file, maybe 'used by another process', exceeded max attempt")
	ERROR_STAT__IS_FOLDER_FUNC__EXCEEDED_MAX_ATTEMPT     = errors.New("Err of stat file, exceeded max attempt")
	ERROR_STAT__GET_MOD_TIME_FUNC__EXCEEDED_MAX_ATTEMPT  = errors.New("Err of stat file, exceeded max attempt")
	ERROR__HASH_WRITE_METHOD__                           = errors.New("Err write hash folder")
	ERROR__HASH_IO_COPY_METHOD__                         = errors.New("Err io copy hash")
)

// GetModTime. Получение времения последней модификации
func GetModTime(log *logrus.Logger, path string) int64 {

	log.Debug(fmt.Sprintf("[utils.GetModTime()] path: %s;", path))

	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Error(fmt.Errorf("[utils.GetModTime()] err: %w, fileName: %s;", err, path))
		log.Error(fmt.Errorf("%w (%s);", ERROR_STAT__GET_MOD_TIME_FUNC__EXCEEDED_MAX_ATTEMPT, err.Error()))
		return 0
	}
	
	modTime := fileInfo.ModTime().UTC().UnixMicro()

	log.Debug(fmt.Sprintf("[utils.GetModTime()] path: %s, modTime: %d;", path, modTime))

	return modTime
}

// IsFolder. Показывает, является ли путь папкой или нет
func IsFolder(log *logrus.Logger, path string) bool {

	log.Debug(fmt.Sprintf("[utils.IsFolder()] path: %s;", path))

	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Error(fmt.Errorf("[utils.IsFolder()] err: %w, fileName: %s;", err, path))
		log.Error(fmt.Errorf("%w (%s);", ERROR_STAT__IS_FOLDER_FUNC__EXCEEDED_MAX_ATTEMPT, err.Error()))
		return false
	}
	isFolder := fileInfo.IsDir()

	log.Debug(fmt.Sprintf("[utils.IsFolder()] path: %s, isFolder: %v;", path, isFolder))

	return isFolder
}

// GetHash. Возвращает хеш файла или папки
func GetHash(log *logrus.Logger, path string) string {

	log.Debug(fmt.Sprintf("[utils.GetHash()] path: %s;", path))

	f, err := os.Open(path)
	if err != nil {
		log.Error(fmt.Errorf("[utils.GetHash()] err: %w, fileName: %s,;", err, path))
		log.Error(fmt.Errorf("[utils.GetHash()] %w (%s);", ERROR_OPEN_FILE__GET_HASH_FUNC__EXCEEDED_MAX_ATTEMPT, err.Error()))
		return ""
	}
	defer f.Close()

	isFolder := IsFolder(log, path)

	if isFolder {
		h := sha256.New()
		mt := GetModTime(log, path)

		_, err = h.Write([]byte(strconv.Itoa(int(mt))))
		if err != nil {
			log.Error(fmt.Errorf("[utils.GetHash()] err: %w, mt: %d;", err, mt))
			log.Error(fmt.Errorf("%w (%s);", ERROR__HASH_WRITE_METHOD__, err.Error()))
			return ""
		}

		hash := hex.EncodeToString(h.Sum(nil))

		log.Debug(fmt.Sprintf("[utils.GetHash()] folder: %s, hash: %s", path, hash))

		return hash
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Error(fmt.Errorf("[utils.writeForHashFolder()] err: %w;", err))
		log.Error(fmt.Errorf("%w (%s);", ERROR__HASH_IO_COPY_METHOD__, err.Error()))
		return ""
	}

	hash := hex.EncodeToString(h.Sum(nil))

	log.Debug(fmt.Sprintf("[utils.GetHash()] file: %s, hash: %s", path, hash))

	return hash
}
