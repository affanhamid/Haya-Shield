package filter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
  "net"

	"github.com/affanhamid/domain-tracker/internal/guardian"
	"github.com/affanhamid/domain-tracker/internal/utils"
)

func LoadBlockedList(path string) map[string]bool {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read blocklist: %v", err)
		return map[string]bool{}
	}

	guardian.UpdateBuffer(path, data)

	var parsed struct {
		Blocked []string `json:"blocked"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		log.Fatalf("Invalid JSON in blocklist: %v", err)
	}

	blocked := make(map[string]bool)
	for _, entry := range parsed.Blocked {
		blocked[strings.TrimSpace(entry)] = true
	}
	return blocked
}

const (
	pfRuleFile = "configs/pf.rules"
	anchorName = "haya.shield"
)

func BlockIP(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	isIPv6 := parsed.To4() == nil
	proto := "inet"
	if isIPv6 {
		proto = "inet6"
	}

	ruleTCP := fmt.Sprintf("block drop quick %s proto tcp from any to %s port 443", proto, ip)
	ruleUDP := fmt.Sprintf("block drop quick %s proto udp from any to %s port 443", proto, ip)
	ruleICMP := fmt.Sprintf("block drop quick %s proto icmp from any to %s", proto, ip)

	newRules := []string{ruleTCP, ruleUDP, ruleICMP}
	existingRules := make(map[string]bool)

	// Step 1: Read existing rules
	file, err := os.Open(utils.GetPath(pfRuleFile))
	if err != nil {
		if os.IsNotExist(err) {
			// Create file with all rules
			content := strings.Join(newRules, "\n") + "\n"
			if err := os.WriteFile(pfRuleFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create pf.rules: %v", err)
			}
			fmt.Printf("📦 Created pf.rules with rules for %s (%s)\n", ip, proto)
			return ReloadPF()
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		existingRules[strings.TrimSpace(scanner.Text())] = true
	}

	// Step 2: Append only new rules
	toAdd := []string{}
	for _, rule := range newRules {
		if !existingRules[rule] {
			toAdd = append(toAdd, rule)
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(utils.GetPath(pfRuleFile), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open pf.rules for appending: %v", err)
	}
	defer f.Close()

	for _, rule := range toAdd {
		if _, err := f.WriteString(rule + "\n"); err != nil {
			return fmt.Errorf("failed to write rule: %v", err)
		}
	}

	fmt.Printf("📦 Appended %d new rules for %s (%s)\n", len(toAdd), ip, proto)
	return ReloadPF()
}

func ReloadPF() error {
	fmt.Println("🔁 Reloading pf.rules...")

	cmd := exec.Command("sudo", "pfctl", "-a", anchorName, "-f", utils.GetPath(pfRuleFile))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("❌ Failed to reload pf.rules: %v\n", err)
	} else {
		fmt.Println("✅ Successfully reloaded pf.rules")
	}
	return err
}
