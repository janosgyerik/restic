package dump

import (
	"archive/zip"
	"context"
	"io"
	"path/filepath"

	"github.com/restic/restic/internal/bloblru"
	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/restic"
)

type zipDumper struct {
	cache *bloblru.Cache
	w     *zip.Writer
}

// Statically ensure that zipDumper implements dumper.
var _ dumper = &zipDumper{}

// WriteZip will write the contents of the given tree, encoded as a zip to the given destination.
func WriteZip(ctx context.Context, repo restic.Repository, tree *restic.Tree, rootPath string, dst io.Writer) error {
	dmp := &zipDumper{
		cache: NewCache(),
		w:     zip.NewWriter(dst),
	}
	return writeDump(ctx, repo, tree, rootPath, dmp)
}

func (dmp *zipDumper) Close() error {
	return dmp.w.Close()
}

func (dmp *zipDumper) dumpNode(ctx context.Context, node *restic.Node, repo restic.Repository) error {
	relPath, err := filepath.Rel("/", node.Path)
	if err != nil {
		return err
	}

	header := &zip.FileHeader{
		Name:               filepath.ToSlash(relPath),
		UncompressedSize64: node.Size,
		Modified:           node.ModTime,
	}
	header.SetMode(node.Mode)

	if IsDir(node) {
		header.Name += "/"
	}

	w, err := dmp.w.CreateHeader(header)
	if err != nil {
		return errors.Wrap(err, "ZipHeader")
	}

	if IsLink(node) {
		if _, err = w.Write([]byte(node.LinkTarget)); err != nil {
			return errors.Wrap(err, "Write")
		}

		return nil
	}

	return WriteNodeData(ctx, w, repo, node, dmp.cache)
}
