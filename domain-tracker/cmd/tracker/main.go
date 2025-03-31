package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
  "strings"

	"github.com/affanhamid/domain-tracker/internal/capture"
	"github.com/affanhamid/domain-tracker/internal/filter"
	"github.com/affanhamid/domain-tracker/internal/guardian"
	"github.com/affanhamid/domain-tracker/internal/utils"
)

var (
	restartChan      = make(chan struct{})
	currentInterface string
)

const launchdPlistPath = "/Library/LaunchDaemons/com.haya.guardian.plist"

func runSnifferLoop() {
	for {
		blocklist := filter.LoadBlockedList(utils.GetPath("configs/blocked_ips.json"))

		done := make(chan struct{})
		go func(device string) {
			err := capture.StartSniffing(device, blocklist, done)
			if err != nil {
				log.Println("❌ Sniffer error:", err)
			}
		}(currentInterface)

		<-restartChan
		fmt.Println("🔁 Restarting sniffer...")
		done <- struct{}{}
		time.Sleep(1 * time.Second)
	}
}

func launchedByLaunchd() bool {
	ppid := syscall.Getppid()
	return ppid == 1
}

func restartLaunchdDaemon(plistPath string) error {
	cmd := exec.Command("launchctl", "bootstrap", "system", plistPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func watchParentAndRecover(plistPath string) {
	parentPidStr := os.Getenv("HAYA_PARENT_PID")
	if parentPidStr == "" {
		return
	}

	parentPid, err := strconv.Atoi(parentPidStr)
	if err != nil {
		log.Println("❌ Invalid parent PID:", err)
		return
	}

	for {
		err := syscall.Kill(parentPid, 0)
		if err != nil {
			fmt.Println("🧨 Parent process is gone.")

			if !launchedByLaunchd() {
				fmt.Println("🔁 Relaunching LaunchDaemon...")
				err := restartLaunchdDaemon(plistPath)
				if err != nil {
					log.Println("❌ Failed to restart LaunchDaemon:", err)
				} else {
					fmt.Println("✅ Relaunched via launchd. Exiting...")
					os.Exit(0)
				}
			}

			os.Exit(1)
		}

		time.Sleep(2 * time.Second)
	}
}

func runHayaShield() {
	go watchParentAndRecover(launchdPlistPath)
	guardian.MonitorFiles()

	currentInterface = capture.AutoDetectInterface()
	fmt.Println("🌐 Initial interface:", currentInterface)

	go runSnifferLoop()

	err := guardian.WatchFile(utils.GetPath("configs/blocked_ips.json"), func() {
		fmt.Println("🛠️ Config changed — reloading blocklist + restarting sniffer")
		restartChan <- struct{}{}
	})
	if err != nil {
		log.Fatal("Watcher failed (blocked_ips.json):", err)
	}

	err = guardian.WatchFile(utils.GetPath("configs/pf.rules"), func() {
		fmt.Println("🛠️ Config changed — reloading pf.rules + restarting sniffer")
		data, err := filter.LoadFileAndCache(utils.GetPath("configs/pf.rules"))
		if err != nil {
			log.Println("❌ Failed to reload pf.rules into buffer:", err)
		} else {
			guardian.UpdateBuffer(utils.GetPath("configs/pf.rules"), data)
		}
		restartChan <- struct{}{}
	})
	if err != nil {
		log.Fatal("Watcher failed (pf.rules):", err)
	}

	err = guardian.WatchFile("/private/etc/pf.conf", func() {
		fmt.Println("🔍 pf.conf changed — checking for anchor lines")
		err := guardian.EnsurePfAnchor()
		if err != nil {
			log.Println("❌ Failed to ensure pf.conf anchor:", err)
		} else {
			restartChan <- struct{}{}
		}
	})
	if err != nil {
		log.Fatal("Watcher failed (pf.conf):", err)
	}

	go func() {
		for {
			newInterface := capture.AutoDetectInterface()
			if newInterface != currentInterface {
				fmt.Printf("🌐 Interface changed: %s → %s\n", currentInterface, newInterface)
				currentInterface = newInterface
				restartChan <- struct{}{}
			}
			time.Sleep(5 * time.Second)
		}
	}()


	go func() {
		for {
			out, err := exec.Command("launchctl", "list").Output()
			if err == nil && !strings.Contains(string(out), "com.haya.guardian") {
				fmt.Println("🛡️ Guardian missing — restoring")
				cmd := exec.Command("launchctl", "bootstrap", "system", "/Library/LaunchDaemons/com.haya.guardian.plist")
				if err := cmd.Run(); err != nil {
					fmt.Println("❌ Failed to restore guardian:", err)
          exec.Command("bash", "/usr/local/bin/haya-intervene.sh").Run()

				} else {
					fmt.Println("✅ Guardian relaunched successfully")
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()

	select {}
}

func main() {
	if os.Getenv("HAYA_CHILD") == "" {
		for {
			cmd := exec.Command(os.Args[0])
			cmd.Env = append(os.Environ(),
				"HAYA_CHILD=1",
				fmt.Sprintf("HAYA_PARENT_PID=%d", os.Getpid()),
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			fmt.Println("🧠 Launching child process...")
			err := cmd.Run()
			fmt.Println("⚠️ Child exited. Restarting in 2s. Reason:", err)

			time.Sleep(2 * time.Second)
		}
	}


	runHayaShield()
}
