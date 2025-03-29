package main

import (
	"fmt"
	"log"
  "time"

	"github.com/affanhamid/domain-tracker/internal/capture"
	"github.com/affanhamid/domain-tracker/internal/filter"
	"github.com/affanhamid/domain-tracker/internal/guardian"
	"github.com/affanhamid/domain-tracker/internal/utils"
)

var restartChan = make(chan struct{})

func runSnifferLoop() {
	for {
		blocklist := filter.LoadBlockedList(utils.GetPath("configs/blocked_ips.json"))
		device := capture.AutoDetectInterface()

		done := make(chan struct{})
		go func() {
			err := capture.StartSniffing(device, blocklist, done)
			if err != nil {
				log.Println("❌ Sniffer error:", err)
			}
		}()

		// Wait for restart signal
		<-restartChan
		fmt.Println("🔁 Restarting sniffer...")
		done <- struct{}{} // tell sniffer to stop
		time.Sleep(1 * time.Second)
	}
}

func main() {
	guardian.MonitorFiles()

	go runSnifferLoop()

	err := guardian.WatchFile(utils.GetPath("configs/blocked_ips.json"), func() {
		fmt.Println("🛠️ Config changed — reloading blocklist + restarting sniffer")
		restartChan <- struct{}{}
	})
	if err != nil {
		log.Fatal("Watcher failed:", err)
	}

	select {} // block forever
}
