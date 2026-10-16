// Command decoy is a terminal EPUB reader with a boss key.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/arkurl/decoy/internal/app"
	"github.com/arkurl/decoy/internal/config"
	"github.com/arkurl/decoy/internal/epub"
	"github.com/arkurl/decoy/internal/storage"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "用法: decoy <book.epub>")
		fmt.Fprintln(os.Stderr, "  摸鱼：按 b 切到伪装界面，再按 b 或 ctrl+b 切回。")
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	book, err := epub.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "打开失败:", err)
		os.Exit(1)
	}
	defer book.Close()

	cfg := config.Load()

	progressPath := storage.DefaultPath()
	store, err := storage.Load(progressPath)
	if err != nil {
		store = storage.NewEmpty(progressPath)
	}

	final, err := tea.NewProgram(app.New(book, cfg, store)).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "运行出错:", err)
		os.Exit(1)
	}
	// Persist the last position even when the program was stopped by a signal
	// (terminal closed, kill) rather than by our own quit handler.
	if fm, ok := final.(app.Model); ok {
		store.Set(book.Path, book.Title, fm.Position())
	}
	_ = store.Save()
}
