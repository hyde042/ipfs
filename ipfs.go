package ipfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"strings"
	"sync"
	"time"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/node"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/ipfs/go-path"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/ipfs/interface-go-ipfs-core/options"
)

var loadPluginsOnce sync.Once

type FS struct {
	n *core.IpfsNode
}

func New(ctx context.Context, repoRoot string) (*FS, error) {
	loadPluginsOnce.Do(func() {
		if err := loadPlugins(); err != nil {
			panic("ipfs: load plugins: " + err.Error())
		}
	})
	if !fsrepo.IsInitialized(repoRoot) {
		conf, err := newDefaultConfig()
		if err != nil {
			return nil, fmt.Errorf("create config: %v", err)
		}
		if err := fsrepo.Init(repoRoot, conf); err != nil {
			return nil, fmt.Errorf("init repo: %v", err)
		}
	}
	r, err := fsrepo.Open(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("open repo: %v", err)
	}
	n, err := core.NewNode(ctx, &node.BuildCfg{
		Repo:      r,
		Permanent: true,
		Online:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("create node: %v", err)
	}
	return &FS{n: n}, nil
}

func loadPlugins() error {
	plugins, err := loader.NewPluginLoader("")
	if err != nil {
		return err
	}
	if err := plugins.Initialize(); err != nil {
		return err
	}
	return plugins.Inject()
}

func newDefaultConfig() (*config.Config, error) {
	identity, err := config.CreateIdentity(os.Stdout, []options.KeyGenerateOption{
		options.Key.Type(options.Ed25519Key),
	})
	if err != nil {
		return nil, fmt.Errorf("create identity: %v", err)
	}
	return config.InitWithIdentity(identity)
}

func (f *FS) Open(name string) (fs.File, error) {
	p := path.FromString(name)

	// TODO: IPNS resolution

	c, _, err := f.n.Resolver.ResolveToLastNode(f.n.Context(), p)
	if err != nil {
		return nil, fmt.Errorf("path resolve: %v", err)
	}
	fn, err := f.n.DAG.Get(f.n.Context(), c)
	if err != nil {
		return nil, fmt.Errorf("dag get: %v", err)
	}
	n, err := unixfile.NewUnixfsFile(f.n.Context(), f.n.DAG, fn)
	if err != nil {
		return nil, fmt.Errorf("create unix file: %v", err)
	}
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
