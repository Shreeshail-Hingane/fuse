package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"golang.org/x/net/context"
)

func main() {
	flag.Parse() //Parses the command line flags from os.Args[1:] and stores it in Flag.Args

	//why is usage needed
	if flag.NArg() != 1 {
		fmt.Println("Error. Wrong number of arguments. Only provide one empty directory")
		os.Exit(2)
	}

	mountpoint := flag.Arg(0)
	c, err := fuse.Mount( //fuse.Mount returns a connection to the mounted filesystem
		mountpoint,
		fuse.FSName("helloworld"), //These 2 attributes are sufficient for Unix OS
		fuse.Subtype("hellofs"),
	)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close() //close the connection after execution of main

	err = fs.Serve(c, FS{})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report.
	// But if fs.Serve runs for the entirety of the duration()(which happens after mount errors have been checked),
	// then, why do we need this.
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}

}

// FS,Dir and File could have been named anything and only refer to the files/dirs we are creating
// Root, Attr, Lookup etc. are the names of methods in the interfaces defined in Serve, and hence, these cannot be changed

// FS implements the hello world file system.
type FS struct{}

func (FS) Root() (fs.Node, error) {
	return Dir{}, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct{}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error { //why is context used here and in other functions
	a.Inode = 1
	a.Mode = os.ModeDir | 0555 //why bitwise or with os.ModeDir. how many bits long is os.ModeDir
	return nil
}

func (Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if name == "hello" {
		return File{}, nil
	} else if name == "folder" {
		return Dir2{}, nil
	}
	return nil, fuse.ENOENT
}

var rootContents = []fuse.Dirent{
	{Inode: 2, Name: "hello", Type: fuse.DT_File},
	{Inode: 2, Name: "folder", Type: fuse.DT_Dir},
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return rootContents, nil
}

// File implements both Node and Handle for the hello file.
type File struct{}

const greeting = "hello, world\n"

func (File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 2
	a.Mode = 0444
	a.Size = uint64(len(greeting))
	return nil
}

func (File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(greeting), nil
}

// Dir2 implements both Node and Handle for the subdirectory
type Dir2 struct{}

func (Dir2) Attr(ctx context.Context, a *fuse.Attr) error { //why is context used here and in other functions
	a.Inode = 3
	a.Mode = os.ModeDir | 0555 //why bitwise or with os.ModeDir. how many bits long is os.ModeDir
	return nil
}

func (Dir2) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if name == "again" {
		return File2{}, nil
	}
	return nil, fuse.ENOENT
}

var dirContents = []fuse.Dirent{
	{Inode: 4, Name: "again", Type: fuse.DT_File},
}

func (Dir2) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return dirContents, nil
}

//File2 implements the file inside the directory named "Folder"

type File2 struct{}

const greeting2 = "hello, again\n"

func (File2) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 4
	a.Mode = 0444
	a.Size = uint64(len(greeting))
	return nil
}

func (File2) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(greeting2), nil
}
