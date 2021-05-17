package ipfs_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/hyde042/ipfs"
)

func TestIPFS(t *testing.T) {
	fsys, err := ipfs.New(context.Background(), "temp")
	if err != nil {
		t.Fatal(err)
	}
	if err := fstest.TestFS(fsys, "readme"); err != nil {
		t.Fatal(err)
	}
}
