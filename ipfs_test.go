package ipfs_test

import (
	"context"
	"io"
	"testing"

	"github.com/hyde042/ipfs"
)

func TestIPFS(t *testing.T) {
	fsys, err := ipfs.New(context.Background(), "temp")
	if err != nil {
		t.Fatal(err)
	}
	/*
		if err := fstest.TestFS(fsys, "readme"); err != nil {
			t.Fatal(err)
		}
	*/
	f, err := fsys.Open("ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data)) // TODO: do some actual validation
}
