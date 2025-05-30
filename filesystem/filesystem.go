// Package filesystem provides the FileSystem interface and its implementations for smarterr.
package filesystem

import (
	"io/fs"
	"os"
)

// FileSystem defines an interface for filesystem operations, including file existence checks.
type FileSystem interface {
	Open(name string) (fs.File, error)
	ReadFile(name string) ([]byte, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Exists(name string) bool
}

// WrappedFS implements FileSystem for a generic fs.FS.
type WrappedFS struct {
	FS fs.FS
}

func NewWrappedFS(root string) *WrappedFS {
	return &WrappedFS{
		FS: os.DirFS(root),
	}
}

func (d *WrappedFS) Open(name string) (fs.File, error) {
	return d.FS.Open(name)
}

func (d *WrappedFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(d.FS, name)
}

func (d *WrappedFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(d.FS, root, fn)
}

// Exists checks if a file exists in the wrapped filesystem.
func (d *WrappedFS) Exists(path string) bool {
	f, err := d.FS.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	stat, err := f.Stat()
	return err == nil && !stat.IsDir()
}
