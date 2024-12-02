package backup

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type Archiver interface {
	Archive(src, dest string) error
}

type zipper struct{}

var ZIP Archiver = (*zipper)(nil)

func (z *zipper) Archive(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), fs.ModePerm); err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}
		//当該パスのファイルを開く
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		//当該パスのzipファイルを作成
		f, err := w.Create(path)
		if err != nil {
			return err
		}
		//zipファイルに元ファイルの内容をコピー
		io.Copy(f, in)
		return nil
	})
}
