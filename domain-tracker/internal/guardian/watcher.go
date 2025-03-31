package guardian

import (
  "os"
  "strings"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

var pfAnchorLines = []string{
	`anchor "haya.shield"`,
	`load anchor "haya.shield" from "/Users/affanhamid/Projects/Haya-Shield/domain-tracker/configs/pf.rules"`,
}

func EnsurePfAnchor() error {
	path := "/private/etc/pf.conf"

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read pf.conf: %w", err)
	}

	content := string(data)
	changed := false

	for _, line := range pfAnchorLines {
		if !strings.Contains(content, line) {
			fmt.Println("➕ Missing anchor line, adding:", line)
			content += "\n" + line
			changed = true
		}
	}

	if changed {
		err = os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("failed to update pf.conf: %w", err)
		}
		fmt.Println("✅ pf.conf updated with missing anchor lines")
	}

	return nil
}

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

        if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
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
