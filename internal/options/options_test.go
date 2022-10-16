package options

import (
	"context"
	"log"
	"testing"
)

func TestEncode(t *testing.T) {
	
	ctx := context.TODO()
	log := log.Logger{}
	opt := Options{
		FilePath: "hello world",
		CurrentOffset: "4096",
		Index: "17",
		Buffer: "1024",
	}


	EncodeOptions(ctx, &log, &opt)
	opt1 := opt
	if opt.Err != nil {
		t.Error(opt.Err)
	}

	DecodeOptions(ctx, &log, &opt)
	opt2 := opt
	if opt.Err != nil {
		t.Error(opt.Err)
	}

	if opt1.FilePath != opt2.FilePath {
		t.Fail()
	}
	if opt1.Buffer != opt2.Buffer {
		t.Fail()
	}
	if opt1.CurrentOffset != opt2.CurrentOffset {
		t.Fail()
	}
	if opt1.Index != opt2.Index {
		t.Fail()
	}
}