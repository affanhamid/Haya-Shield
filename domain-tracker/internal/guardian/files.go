package guardian

import (
	"fmt"
	"os"
	"sync"
	"time"
  "path/filepath"

	"github.com/affanhamid/domain-tracker/internal/utils"
)

type FileCache struct {
	Path    string
	Default string
	Mutex   sync.RWMutex
	Buffer  []byte
}


var files = []*FileCache{
	{
		Path:    "configs/blocked_ips.json",
		Default: `{"blocked": []}`,
	},
	{
		Path:    "configs/pf.rules",
		Default: `# Haya Shield rules\n`,
	},
}

func MonitorFiles() {
	for _, f := range files {
    f.preload()
		go f.watch()
	}
}

// Watch for file deletion and restore from in-memory buffer or fallback
func (f *FileCache) watch() {
	for {
		if _, err := os.Stat(utils.GetPath(f.Path)); os.IsNotExist(err) {
			fmt.Println("🚨 File deleted:", utils.GetPath(f.Path))

			f.Mutex.RLock()
			data := f.Buffer
			f.Mutex.RUnlock()

			if len(data) == 0 {
				data = []byte(f.Default)
			}
      // 🛠 Ensure parent directory exists
			dir := filepath.Dir(utils.GetPath(f.Path))
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Println("❌ Failed to create directory:", err)
				continue
			}

			if err := os.WriteFile(utils.GetPath(f.Path), data, 0644); err != nil {
				fmt.Println("❌ Failed to restore file:", err)
			} else {
				fmt.Println("✅ Restored from memory:", utils.GetPath(f.Path))
				UpdateBuffer(utils.GetPath(f.Path), data) // sync memory with disk
			}
		}
		time.Sleep(2 * time.Second)
	}
}

// Manually update the memory buffer when writing or loading the file
func UpdateBuffer(path string, content []byte) {
	for _, f := range files {
		if utils.GetPath(f.Path) == path {
			f.Mutex.Lock()
			f.Buffer = content
			f.Mutex.Unlock()
			return
		}
	}
}

func (f *FileCache) preload() {
	fullPath := utils.GetPath(f.Path)

	data, err := os.ReadFile(fullPath)
	if err == nil && len(data) > 0 {
		f.Mutex.Lock()
		f.Buffer = data
		f.Mutex.Unlock()
		fmt.Println("📦 Preloaded:", fullPath)
	}
}

// Unified write function to disk + memory
func WriteAndCache(path string, content []byte) error {
	if err := os.WriteFile(path, content, 0644); err != nil {
		return err
	}
	UpdateBuffer(path, content)
	return nil
}
