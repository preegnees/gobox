package errors

import (
	"errors"
)

var (
	ERROR__GET_ALL_FILES_FROM_DIR__ = errors.New("Err get files from folder (ioutil.ReadDir)")
	ERROR__WILL_CAUSE_A_STOP__      = errors.New("Err will cause a stop")
	ERROR__GET_METADATA__ = errors.New("err get metadata")
)
