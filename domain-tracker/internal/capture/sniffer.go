
package capture

import (
	"fmt"
	"log"
	"strings"

	"github.com/affanhamid/domain-tracker/internal/utils"
	"github.com/affanhamid/domain-tracker/internal/filter"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

func AutoDetectInterface() string {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal("Error finding devices:", err)
	}

	for _, dev := range devices {
		if strings.HasPrefix(dev.Name, "utun") {
			for _, addr := range dev.Addresses {
				if ip := addr.IP.To4(); ip != nil {
					return dev.Name
				}
			}
		}
	}
	for _, dev := range devices {
		for _, addr := range dev.Addresses {
			if ip := addr.IP.To4(); ip != nil && !ip.IsLoopback() {
				return dev.Name
			}
		}
	}
	log.Fatal("No suitable interface found.")
	return ""
}

func StartSniffing(device string, blocklist map[string]bool, stop <-chan struct{}) error {
	handle, err := pcap.OpenLive(device, 65536, true, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()

	err = handle.SetBPFFilter("ip")
	if err != nil {
		return err
	}

	fmt.Println("📡 Sniffing on", device, "...")

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		select {
		case <-stop:
			fmt.Println("🛑 Sniffer stopped.")
			return nil
		case packet := <-packetSource.Packets():
			if netLayer := packet.NetworkLayer(); netLayer != nil {
				_, dst := netLayer.NetworkFlow().Endpoints()
				dstIP := dst.String()

				if utils.IsPrivateIP(dstIP) {
					continue
				}
			if blocklist[dstIP] {
        fmt.Println("🚫 BLOCKED IP:", dstIP)
        err := filter.BlockIP(dstIP)
          if err != nil {
            fmt.Println("❌ Failed to firewall IP:", err)
          }
        }
			}
		}
  }
}
