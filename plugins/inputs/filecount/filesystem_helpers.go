package filecount

import (
	"errors"
	"io"
	"os"
	"time"
)

/*
	The code below is lifted from numerous articles and originates from Andrew Gerrand's 10
	things you (probably) don't know about Go. it allows for mocking a filesystem; this allows
	for consistent testing of this code across platforms (directory sizes reported
	differently by different platforms, for example), while preserving the rest of
	the functionality as-is, without modification.
*/

type fileSystem interface {
	Open(name string) (file, error)
	Stat(name string) (os.FileInfo, error)
}

type file interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	Stat() (os.FileInfo, error)
}

// osFS implements fileSystem using the local disk
type osFS struct{}

func (osFS) Open(name string) (file, error)        { return os.Open(name) }
func (osFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

/*
	The following are for mocking the filesystem - this allows us to mock Stat() files. This means that we can set file attributes, and know that they
	will be the same regardless of the platform sitting underneath our tests (directory sizes vary)

	NOTE: still need the on-disk file structure to mirror this because the 3rd party library ("github.com/karrick/godirwalk") uses its own
	walk functions, that we cannot mock from here.
*/

// When tests can be done on windows, remove these nolint:unused comments - see filesystem_helpers_test.go
//nolint:unused
type fakeFileSystem struct {
	files map[string]fakeFileInfo
}

//nolint:unused
type fakeFileInfo struct {
	name     string
	size     int64
	filemode uint32
	modtime  time.Time
	isdir    bool
	sys      interface{}
}

func (f fakeFileInfo) Name() string       { return f.name }                  //nolint:unused
func (f fakeFileInfo) Size() int64        { return f.size }                  //nolint:unused
func (f fakeFileInfo) Mode() os.FileMode  { return os.FileMode(f.filemode) } //nolint:unused
func (f fakeFileInfo) ModTime() time.Time { return f.modtime }               //nolint:unused
func (f fakeFileInfo) IsDir() bool        { return f.isdir }                 //nolint:unused
func (f fakeFileInfo) Sys() interface{}   { return f.sys }                   //nolint:unused

func (f fakeFileSystem) Open(name string) (file, error) { //nolint:unused
	return nil, &os.PathError{Op: "Open", Path: name, Err: errors.New("Not implemented by fake filesystem")}
}

func (f fakeFileSystem) Stat(name string) (os.FileInfo, error) { //nolint:unused
	if fakeInfo, found := f.files[name]; found {
		return fakeInfo, nil
	}
	return nil, &os.PathError{Op: "Stat", Path: name, Err: errors.New("No such file or directory")}

}
