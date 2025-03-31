package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	daemonLabel   = "com.haya.shield"
	plistPath     = "/Library/LaunchDaemons/com.haya.shield.plist"
	checkInterval = 10 * time.Second
)

func isDaemonRunning() bool {
	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		fmt.Println("❌ Failed to run launchctl list:", err)
		return false
	}
	return strings.Contains(string(out), daemonLabel)
}

func restoreDaemon() error {
	fmt.Println("🛠️ Attempting to restore LaunchDaemon...")
	cmd := exec.Command("launchctl", "bootstrap", "system", plistPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func main() {
	fmt.Println("👁️  Haya Guardian started")

	for {
		if !isDaemonRunning() {
			fmt.Println("🚨 Haya Shield daemon missing — trying to restore")
			if err := restoreDaemon(); err != nil {
				fmt.Println("❌ Failed to restore daemon:", err)
			} else {
				fmt.Println("✅ Daemon relaunched successfully")
			}
		}
		time.Sleep(checkInterval)
	}
}
