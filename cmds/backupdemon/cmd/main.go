package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matryer/filedb"
	"github.com/taimats/butool/cmds/backup"
)

type path struct {
	Path string
	Hash string
}

func main() {
	//エラー共通処理
	var fatalErr error
	defer func() {
		if fatalErr != nil {
			log.Fatalln(fatalErr)
		}
	}()

	//flagの実装
	var (
		interval = flag.Duration("interval", 10*time.Second, "チェック間隔")
		archive  = flag.String("archive", "archive", "アーカイブの保存先")
		dbpath   = flag.String("db", "./db", "filedbのデータ保存先")
	)
	flag.Parse()

	//monitorインスタンスの生成
	m := &backup.Monitor{
		Destination: *archive,
		Archiver:    backup.ZIP,
		Paths:       make(map[string]string),
	}

	//ファイルデータベースへのアクセス
	db, err := filedb.Dial(*dbpath)
	if err != nil {
		fatalErr = err
		return
	}
	defer db.Close()

	//json形式でデータを取得
	col, err := db.C("paths")
	if err != nil {
		fatalErr = err
		return
	}

	//json形式からpath構造体にマッピングし、キャッシュ代わりを生成
	var path path
	col.ForEach(func(i int, b []byte) bool {
		if err := json.Unmarshal(b, &path); err != nil {
			fatalErr = err
			return true
		}
		m.Paths[path.Path] = path.Hash
		return false
	})

	if len(m.Paths) < 1 {
		fatalErr = errors.New("パスがありません。backupツールを使って追加して下さい。")
		return
	}

	//バックグラウンドで実行する処理
	check(m, col)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
LOOP:
	for {
		select {
		case <-time.After(*interval):
			check(m, col)
		case <-signalChan:
			fmt.Println()
			log.Println("終了します")
			break LOOP
		}
	}
}

func check(m *backup.Monitor, col *filedb.C) {
	log.Println("チェックを開始します")
	counter, err := m.Now()
	if err != nil {
		log.Panicln("バックアップに失敗しました:", err)
	}

	if counter > 0 {
		log.Printf("%d個のディレクトリをアーカイブしました\n", counter)

		var path path
		col.SelectEach(func(_ int, b []byte) (bool, []byte, bool) {
			if err := json.Unmarshal(b, &path); err != nil {
				log.Println("JSONデータの読み込みに失敗しました。次の項目に進みます:", err)
				return true, b, false
			}
			path.Hash, _ = m.Paths[path.Path]
			newdata, err := json.Marshal(&path)
			if err != nil {
				log.Println("JSONデータの書き出しに失敗しました。次の項目に進みます:", err)
				return true, b, false
			}
			return true, newdata, false
		})
	} else {
		log.Println("変更はありません")
	}
}
