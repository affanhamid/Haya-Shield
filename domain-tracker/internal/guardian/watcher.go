package guardian

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func WatchFile(path string, onChange func()) error {
	dir := filepath.Dir(path)
	filename := filepath.Base(path)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.Add(dir)
	if err != nil {
		return err
	}

	fmt.Printf("👀 Watching %s for changes to %s\n", dir, filename)

	go func() {
		var lastTrigger time.Time

		for {
			select {
			case event := <-watcher.Events:
				if filepath.Base(event.Name) != filename {
					continue
				}

				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					if time.Since(lastTrigger) > time.Second {
						lastTrigger = time.Now()
						fmt.Println("🔁 File changed:", event.Op)
						onChange()
					}
				}

			case err := <-watcher.Errors:
				fmt.Println("❌ Watcher error:", err)
			}
		}
	}()

	return nil
}
