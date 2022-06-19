package main

import (
	"syscall"
    "path/filepath"

    "github.com/google/uuid"
	"github.com/hanwen/go-fuse/v2/fs"
)

type S3Node struct {
    fs.LoopbackNode

    uuid    string      `json: "uuid"`
    isLocal bool        `json: "isLocal"`
    server  string      `json: "server"`
}

func (n *S3Node) path() string {
	path := n.LoopbackNode.Path(n.LoopbackNode.Root())
	return filepath.Join(n.RootData.Path, path)
}

func newNode(rootData *fs.LoopbackRoot, parent *fs.Inode, name string, st *syscall.Stat_t) fs.InodeEmbedder {
    return &S3Node{
        LoopbackNode: fs.LoopbackNode{
            RootData: rootData,
        },
        uuid: uuid.New().String(),
        isLocal: true,
        server: "",
    }
}

func NewLoopbackRoot(rootPath string) (fs.InodeEmbedder, error) {
	var st syscall.Stat_t
	err := syscall.Stat(rootPath, &st)
	if err != nil {
		return nil, err
	}

	root := &fs.LoopbackRoot{
        Path: rootPath,
        Dev:  uint64(st.Dev),
        NewNode: newNode,
	}

	return newNode(root, nil, "", &st), nil
}
