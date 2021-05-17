package ipfs

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sync"

	config "github.com/ipfs/go-ipfs-config"
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
	return fileFromNode(p, n)
}
