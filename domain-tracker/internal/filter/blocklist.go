
package filter

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"
	"bufio"
	"fmt"
	"os"
	"os/exec"

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
	rule := fmt.Sprintf("block drop from any to %s", ip)

	// Step 1: Check if rule already exists
	file, err := os.Open(pfRuleFile)
	if err != nil {
		// If file doesn't exist, create it
		if os.IsNotExist(err) {
			err = os.WriteFile(pfRuleFile, []byte(rule+"\n"), 0644)
			if err != nil {
				return fmt.Errorf("failed to create pf.rules: %v", err)
			}
			return ReloadPF()
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == rule {
			// Already exists
			return nil
		}
	}

	// Step 2: Append rule
	f, err := os.OpenFile(pfRuleFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(rule + "\n"); err != nil {
		return err
	}

	// Step 3: Reload into anchor
	return ReloadPF()
}

func ReloadPF() error {
	cmd := exec.Command("sudo", "pfctl", "-a", anchorName, "-f", pfRuleFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
