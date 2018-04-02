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
	original := flag.Arg(0) //original is the directory to be mirrored
	mountpoint := flag.Arg(1)

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("completeFS"),
		fuse.Subtype("writeFS"),
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

//FSroot implements the root of the filesystem--------------------------------------------------------
type FSroot struct {
	path string
}

func (root FSroot) Root() (fs.Node, error) {
	return Dir{root.path}, nil
}

//end of root struct

//Dir implements a directory---------------------------------------------------------------------------
type Dir struct {
	path string
}

func (dir Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	file_stat := syscall.Stat_t{}      //Stat_t struct holds info of a file(dir is also a file in some sense)
	syscall.Stat(dir.path, &file_stat) //Stat() function fills the attributes of the file specified by "path"
	a.Inode = uint64(file_stat.Ino)
	a.Mode = os.ModeDir | os.FileMode(file_stat.Mode)
	a.Size = uint64(file_stat.Size)
	return nil
}

func (dir Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	opendir, err := os.Open(dir.path) //return type: *File
	if err != nil {
		log.Fatal(err)
	}
	fileInfoList, err := opendir.Readdir(-1) //return type []fileInfo
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
	opendir, err := os.Open(dir.path) //return os.file type object
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

func (dir Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	err := os.Mkdir(dir.path+"/"+req.Name, req.Mode)
	newdir := Dir{path: dir.path + "/" + req.Name}
	return newdir, err
}

//end of Dir struct-------------------------------------------------------------------------------

//File implements a file--------------------------------------------------------------------------
type File struct {
	path string
}

//A Handle is the interface required of an opened file or directory.
type FileHandle struct {
	path   string
	handle *os.File //os.File represents an open file descriptor
}

func (file File) Attr(ctx context.Context, a *fuse.Attr) error {
	file_info := syscall.Stat_t{}       //Stat_t struct holds info of a file
	syscall.Stat(file.path, &file_info) //Stat() function fills the attributes of the file specified by "path"
	a.Inode = uint64(file_info.Ino)
	a.Mode = os.FileMode(file_info.Mode)
	a.Size = uint64(file_info.Size)
	return nil
}

func (f File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if req.Valid.Size() {
		os.Truncate(f.path, int64(req.Size))
	}
	if req.Valid.Atime() || req.Valid.Mtime() {
		os.Chtimes(f.path, req.Atime, req.Mtime)
	}
	if req.Valid.Gid() || req.Valid.Uid() {
		os.Chown(f.path, int(req.Uid), int(req.Gid))
	}
	if req.Valid.Mode() {
		os.Chmod(f.path, req.Mode)
	}

	return nil
}

/*func (file File) ReadAll(ctx context.Context) ([]byte, error) {
	content, err := ioutil.ReadFile(file.path)
	if err != nil {
		log.Fatal(err)
	}
	return content, nil

}*/

func (fh FileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	txt, err := ioutil.ReadFile(fh.path)
	resp.Data = txt
	return err
}

func (dir Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	openFile, err := os.Create(dir.path + "/" + req.Name)
	file := File{path: dir.path + "/" + req.Name}
	handle := FileHandle{path: dir.path + "/" + req.Name, handle: openFile}
	return file, handle, err
}

func (file File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fileinfo, err := os.Lstat(file.path)
	openFile, err := os.OpenFile(file.path, int(req.Flags), fileinfo.Mode()) //return type *os.File
	handle := FileHandle{path: file.path, handle: openFile}

	resp.Handle = fuse.HandleID(req.Header.Node)
	resp.Flags = fuse.OpenResponseFlags(req.Flags)

	return handle, err
}

func (fh FileHandle) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	n, err := fh.handle.Write(req.Data) //os.Write is method on type os.File
	resp.Size = int(n)
	return err
}

func (fh FileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	err := fh.handle.Close()
	return err
}

func (dir Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	err := os.Remove(dir.path + "/" + req.Name)
	return err
}

//end of File struct-------------------------------------------------------------------------------
