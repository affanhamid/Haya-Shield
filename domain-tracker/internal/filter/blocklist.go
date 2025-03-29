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

	"github.com/affanhamid/domain-tracker/internal/guardian"
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
ruleTCP := fmt.Sprintf("block drop quick proto tcp from any to %s port 443", ip)
ruleICMP := fmt.Sprintf("block drop quick proto icmp from any to %s", ip)

	existingRules := make(map[string]bool)

	// Step 1: Read existing rules
	file, err := os.Open(pfRuleFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist — create with both rules
			content := ruleTCP + "\n" + ruleICMP + "\n"
			if err := os.WriteFile(pfRuleFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create pf.rules: %v", err)
			}
			return ReloadPF()
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		existingRules[strings.TrimSpace(scanner.Text())] = true
	}

	// Step 2: Append missing rules
	var newRules []string
	if !existingRules[ruleTCP] {
		newRules = append(newRules, ruleTCP)
	}
	if !existingRules[ruleICMP] {
		newRules = append(newRules, ruleICMP)
	}

	if len(newRules) == 0 {
		// No new rules to add
		return nil
	}

	f, err := os.OpenFile(pfRuleFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, rule := range newRules {
		if _, err := f.WriteString(rule + "\n"); err != nil {
			return err
		}
	}

	fmt.Println("📦 Added new rules, reloading PF...")
	return ReloadPF()
}

func ReloadPF() error {
	fmt.Println("🔁 Reloading pf.rules...")

	cmd := exec.Command("sudo", "pfctl", "-a", anchorName, "-f", pfRuleFile)
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
