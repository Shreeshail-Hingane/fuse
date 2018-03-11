package testpkg

import (
	"golang.org/x/net/context"
	"testing"
)

var rootHandle FS
var dirHandle Dir
var ctx context.Context

func TestRoot(t *testing.T) {
	err := rootHandle.Root()
	if err != nil {
		t.Error("root returning error")
	}
}

func TestLookup(t *testing.T) {
	err1 := dirHandle.Lookup(ctx, "hello")
	err2 := dirHandle.Lookup(ctx, "welcome")
	if (err1 != nil) || (err2 == nil) {
		t.Error("dir search error")
	}
}
