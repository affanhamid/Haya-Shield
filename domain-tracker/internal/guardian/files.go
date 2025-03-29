package guardian

import (
	"fmt"
	"os"
	"sync"
	"time"
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
		go f.watch()
	}
}

// Watch for file deletion and restore from in-memory buffer or fallback
func (f *FileCache) watch() {
	for {
		if _, err := os.Stat(f.Path); os.IsNotExist(err) {
			fmt.Println("🚨 File deleted:", f.Path)

			f.Mutex.RLock()
			data := f.Buffer
			f.Mutex.RUnlock()

			if len(data) == 0 {
				data = []byte(f.Default)
			}

			if err := os.WriteFile(f.Path, data, 0644); err != nil {
				fmt.Println("❌ Failed to restore file:", err)
			} else {
				fmt.Println("✅ Restored from memory:", f.Path)
				UpdateBuffer(f.Path, data) // sync memory with disk
			}
		}
		time.Sleep(2 * time.Second)
	}
}

// Manually update the memory buffer when writing or loading the file
func UpdateBuffer(path string, content []byte) {
	for _, f := range files {
		if f.Path == path {
			f.Mutex.Lock()
			f.Buffer = content
			f.Mutex.Unlock()
			return
		}
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
