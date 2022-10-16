package options

import (
	"bytes"
	"context"
	"log"
)

const (
	FILE_PATH_SIZE      = 16
	CURRENT_OFFSET_SIZE = 32
	INDEX_SIZE          = 16
	BUFFER_SIZE         = 8
)

const (
	BOUND     = "\x00\x00\x00\x00"
	DELIMITER = "\x00\x00"
)

type Options struct {
	FilePath      string
	CurrentOffset string
	Index         string
	Buffer        string
	Opt           []byte
	Err           error
}

func EncodeOptions(ctx context.Context, log *log.Logger, o *Options) {

	buf := new(bytes.Buffer)

	data := []string{
		BOUND, 
		o.FilePath, DELIMITER, 
		o.CurrentOffset, DELIMITER, 
		o.Index, DELIMITER, 
		o.Buffer, 
		BOUND,
	}

	for _, v := range data {
		_, err := buf.WriteString(v)
		if err != nil {
			o.Err = err
		}
	}

	o.Opt = buf.Bytes()
}

func DecodeOptions(ctx context.Context, log *log.Logger, o *Options) {

	opts := bytes.Split(bytes.Split(o.Opt, []byte(BOUND))[1], []byte(DELIMITER))
	
	o.FilePath = string(opts[0])
	o.CurrentOffset = string(opts[1])
	o.Index = string(opts[2])
	o.Buffer = string(opts[3])
}
