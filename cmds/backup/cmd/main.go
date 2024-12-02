package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/matryer/filedb"
)

type path struct {
	Path string
	Hash string
}

func (p path) String() string {
	return fmt.Sprintf("%s[%s]", p.Path, p.Hash)
}

func main() {
	var fatalErr error
	defer func() {
		if fatalErr != nil {
			flag.PrintDefaults()
			log.Fatalln(fatalErr)
		}
	}()

	var (
		dbpath = flag.String("db", "./backupdata", "データベースのディレクトリパス")
	)
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fatalErr = errors.New("エラー；フラグを指定してください")
		return
	}

	db, err := filedb.Dial(*dbpath)
	if err != nil {
		fatalErr = err
		return
	}
	db.Close()

	col, err := db.C("paths")
	if err != nil {
		fatalErr = err
		return
	}

	switch strings.ToLower(args[0]) {
	case "list":
		var path path
		col.ForEach(func(i int, b []byte) bool {
			err := json.Unmarshal(b, &path)
			if err != nil {
				fatalErr = err
				return true
			}
			fmt.Printf("= %s\n", path)
			return false
		})
	case "add":
		if len(args[1:]) == 0 {
			fatalErr = errors.New("追加するパスを指定ください。")
			return
		}
		for _, p := range args {
			path := &path{Path: p, Hash: "まだアーカイブされていません"}
			if err = col.InsertJSON(path); err != nil {
				fatalErr = err
				return
			}
			fmt.Printf("= %s\n", path)
		}
	case "remove":
		var path path
		col.RemoveEach(func(i int, b []byte) (bool, bool) {
			err := json.Unmarshal(b, &path)
			if err != nil {
				fatalErr = err
				return false, true
			}

			for _, p := range args[1:] {
				if path.Path == p {
					fmt.Printf("- %s\n", path)
					return true, false
				}
			}
			return false, false
		})

	}
}
