package ipfs

import (
	"errors"
	"io"
	"io/fs"
	pathpkg "path"
	"strings"
	"time"

	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-path"
)

func fileFromNode(p path.Path, n files.Node) (fs.File, error) {
	switch n := n.(type) {
	case files.File:
		return &file{File: n, fileNode: fileNode{
			p: p,
			m: fs.ModePerm,
		}}, nil
	case files.Directory:
		return &dir{Directory: n, fileNode: fileNode{
			p: p,
			m: fs.ModePerm | fs.ModeDir,
		}}, nil
	default:
		return nil, errors.New("invalid file type")
	}
}

type file struct {
	files.File
	fileNode
}

type dir struct {
	files.Directory
	fileNode
}

func (d *dir) Read([]byte) (int, error) { return 0, fs.ErrInvalid }

func (d *dir) ReadDir(n int) ([]fs.DirEntry, error) {
	var (
		dir []fs.DirEntry
		e   = d.Entries()
	)
	for e.Next() {
		var m fs.FileMode
		if strings.HasSuffix(e.Name(), "/") {
			m = fs.ModeDir
		}
		dir = append(dir, dirEntry{
			fileNode{Node: e.Node(), p: path.Path(e.Name()), m: m},
		})
		if n > 0 && len(dir) == n {
			break
		}
	}
	if err := e.Err(); err != nil {
		return nil, err
	}
	if n > 0 && len(dir) < n {
		return dir, io.EOF
	}
	return dir, nil
}

type dirEntry struct{ fileNode }

func (d dirEntry) Name() string               { return pathpkg.Base(string(d.p)) }
func (d dirEntry) IsDir() bool                { return d.m.IsDir() }
func (d dirEntry) Type() fs.FileMode          { return d.m }
func (d dirEntry) Info() (fs.FileInfo, error) { return d.Stat() }

type fileNode struct {
	files.Node
	p path.Path
	m fs.FileMode
}

func (n fileNode) Stat() (fs.FileInfo, error) {
	size, err := n.Size()
	if err != nil {
		return nil, err
	}
	return fileInfo{
		p:    n.p,
		size: size,
		m:    n.m,
	}, nil
}

type fileInfo struct {
	p    path.Path
	size int64
	m    fs.FileMode
}

func (f fileInfo) Name() string       { return pathpkg.Base(string(f.p)) }
func (f fileInfo) Size() int64        { return f.size }
func (f fileInfo) Mode() fs.FileMode  { return f.m }
func (f fileInfo) ModTime() time.Time { return time.Time{} }
func (f fileInfo) IsDir() bool        { return f.m.IsDir() }
func (f fileInfo) Sys() interface{}   { return nil }
