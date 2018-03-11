package testpkg

import (
	"bazil.org/fuse"
	_ "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type FS struct {
	asd int
}

func (FS) Root() error {
	return nil
}

type Dir struct {
	no int
}

func (Dir) Lookup(ctx context.Context, name string) error {
	if name == "hello" {
		return nil
	}
	return fuse.ENOENT
}

type File struct{}

const greeting = "hello, world\n"

func (File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(greeting), nil
}
