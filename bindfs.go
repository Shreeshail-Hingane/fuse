package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"golang.org/x/net/context"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		os.Exit(2)
	}
	original := flag.Arg(0) //original is the string containing the directory name
	mountpoint := flag.Arg(1)

	//filelist, err := folder.Readdirnames(0) //returns list of files in original folder

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("MirrorFS"),
		fuse.Subtype("bindfs"),
		fuse.ReadOnly(),
	)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	myfs := FSroot{original}
	err = fs.Serve(c, myfs)
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

//FS implements the root of the filesystem
type FSroot struct {
	path string
}

func (root FSroot) Root() (fs.Node, error) {
	return Dir{root.path}, nil
}

//Dir implements a directory
type Dir struct {
	path string
}

func (dir Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	file_info := syscall.Stat_t{}      //Stat_t struct holds info of a file(dir is also a file in some sense)
	syscall.Stat(dir.path, &file_info) //Stat() function fills the attributes of the file specified by "path"
	a.Inode = uint64(file_info.Ino)
	a.Mode = os.ModeDir | os.FileMode(file_info.Mode)
	a.Size = uint64(file_info.Size)
	return nil
}

func (dir Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	opendir, err := os.Open(dir.path)
	if err != nil {
		log.Fatal(err)
	}
	fileInfoList, err := opendir.Readdir(-1)
	for _, file := range fileInfoList {
		if file.Name() == name {
			if file.IsDir() {
				return Dir{dir.path + "/" + file.Name()}, nil
			} else {
				return File{dir.path + "/" + file.Name()}, nil
			}
		}
	}
	return nil, fuse.ENOENT
}

func (dir Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fileList := []fuse.Dirent{}
	opendir, err := os.Open(dir.path) //return os.file type object to variable "opendir".
	if err != nil {
		log.Fatal(err)
	}
	fileInfoList, err := opendir.Readdir(-1) //Readdir has return type FileInfo(necessary to get name of files)

	for _, file := range fileInfoList {
		fileStat := syscall.Stat_t{}
		syscall.Stat(dir.path+"/"+file.Name(), &fileStat) //Necessary to get #inode
		if file.IsDir() {
			fileList = append(fileList, fuse.Dirent{Inode: fileStat.Ino, Name: file.Name(), Type: fuse.DT_Dir})
		} else {
			fileList = append(fileList, fuse.Dirent{Inode: fileStat.Ino, Name: file.Name(), Type: fuse.DT_File})
		}
	}
	return fileList, nil

}

//File implements a file
type File struct {
	path string
}

const maxFileSize = 10000000

func (file File) Attr(ctx context.Context, a *fuse.Attr) error {
	file_info := syscall.Stat_t{}       //Stat_t struct holds info of a file
	syscall.Stat(file.path, &file_info) //Stat() function fills the attributes of the file specified by "path"
	a.Inode = uint64(file_info.Ino)
	a.Mode = os.FileMode(file_info.Mode)
	a.Size = uint64(file_info.Size)
	return nil
}

func (file File) ReadAll(ctx context.Context) ([]byte, error) {
	content, err := ioutil.ReadFile(file.path)
	if err != nil {
		log.Fatal(err)
	}
	return content, nil

}
