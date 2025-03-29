package utils

import (
    "os"
    "strings"
    "path/filepath"
)

func GetPath(rel string) string {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	exeDir := filepath.Dir(exePath)

	// dev mode fallback
	if strings.Contains(exePath, "/go-build") {
		// fallback to working directory
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		return filepath.Join(wd, rel)
	}

	return filepath.Join(exeDir, rel)
}
