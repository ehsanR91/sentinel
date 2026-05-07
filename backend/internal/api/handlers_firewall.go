package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// UFWRule represents a parsed UFW rule.
type UFWRule struct {
	Number   int    `json:"number"`
	To       string `json:"to"`
	Action   string `json:"action"`
	From     string `json:"from"`
	Protocol string `json:"protocol"`
	Comment  string `json:"comment"`
}

func (h *Handlers) GetFirewallRules(w http.ResponseWriter, r *http.Request) {
	active := isUFWActive()
	rules, err := parseUFWStatus()
	if err != nil || rules == nil {
		rules = []UFWRule{}
	}

	conns := getActiveConnections()

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":     active,
		"rules":       rules,
		"connections": conns,
	})
}

type addRuleRequest struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Action   string `json:"action"`
	From     string `json:"from"`
	Comment  string `json:"comment"`
}

func (h *Handlers) AddFirewallRule(w http.ResponseWriter, r *http.Request) {
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	var req addRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Port == "" {
		writeError(w, http.StatusBadRequest, "Invalid request: port is required")
		return
	}

	// Validate IP format if 'from' is provided
	if req.From != "" && req.From != "any" {
		if !isValidIPOrCIDR(req.From) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid IP address or CIDR format: %s", req.From))
			return
		}
	}

	action := req.Action
	if action == "" {
		action = "allow"
	}
	// Validate action
	action = strings.ToUpper(action)
	if action != "ALLOW" && action != "DENY" && action != "REJECT" && action != "LIMIT" {
		writeError(w, http.StatusBadRequest, "Invalid action: must be allow, deny, reject, or limit")
		return
	}
	proto := req.Protocol
	if proto == "" {
		proto = "tcp"
	}

	args := []string{strings.ToLower(action)}
	if req.From != "" && req.From != "any" {
		args = append(args, "from", req.From, "to", "any", "port", req.Port)
	} else {
		args = append(args, req.Port+"/"+proto)
	}
	if req.Comment != "" {
		args = append(args, "comment", req.Comment)
	}

	out, err := runUFW(args...)
	if err != nil {
		errMsg := string(out)
		if errMsg == "" {
			errMsg = err.Error()
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add firewall rule: %s", strings.TrimSpace(errMsg)))
		return
	}
	runUFW("reload")
	writeJSON(w, http.StatusOK, map[string]string{"status": "added", "output": string(out)})
}

func (h *Handlers) DeleteFirewallRule(w http.ResponseWriter, r *http.Request) {
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	idStr := chi.URLParam(r, "id")
	ruleNum, err := strconv.Atoi(idStr)
	if err != nil || ruleNum < 1 {
		writeError(w, http.StatusBadRequest, "invalid rule number")
		return
	}

	// --force avoids interactive prompt
	out, err := runUFW("--force", "delete", strconv.Itoa(ruleNum))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ufw delete error: "+string(out))
		return
	}
	runUFW("reload")
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// findUFWBin returns the absolute path to the ufw binary, or "" if not found.
// Searches known install locations so it works regardless of the process PATH
// or sudo's secure_path (ufw moved from /usr/sbin to /usr/bin on Debian 12+/Ubuntu 24.04+).
func findUFWBin() string {
	for _, p := range []string{"/usr/sbin/ufw", "/usr/bin/ufw", "/sbin/ufw", "/bin/ufw"} {
		if info, err := os.Stat(p); err == nil && info.Mode()&0o111 != 0 {
			return p
		}
	}
	if p, err := exec.LookPath("ufw"); err == nil {
		return p
	}
	return ""
}

// runUFW executes ufw with the given args, trying sudo -n first then plain ufw.
// Uses the absolute path to the ufw binary so sudo can match it against sudoers
// regardless of PATH differences between environments.
func runUFW(args ...string) ([]byte, error) {
	ufwBin := findUFWBin()
	if ufwBin == "" {
		return nil, fmt.Errorf("ufw binary not found in any known location")
	}
	sudoArgs := append([]string{"-n", ufwBin}, args...)
	out, err := exec.Command("sudo", sudoArgs...).CombinedOutput()
	if err == nil {
		return out, nil
	}
	// Fallback: try running ufw directly (works if the process already has root or CAP_NET_ADMIN)
	return exec.Command(ufwBin, args...).CombinedOutput()
}

// parseUFWStatus runs `ufw status numbered` (via sudo -n) and parses the output.
// Falls back to parsing /etc/ufw/user.rules if the command fails.
func parseUFWStatus() ([]UFWRule, error) {
	out, err := runUFW("status", "numbered")
	if err != nil {
		return parseUFWRulesFiles()
	}

	if strings.Contains(string(out), "inactive") {
		return []UFWRule{}, nil
	}

	var rules []UFWRule
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Lines look like: [ 1] 22/tcp                     ALLOW IN    Anywhere
		if !strings.HasPrefix(line, "[") {
			continue
		}
		bracket := strings.Index(line, "]")
		if bracket < 0 {
			continue
		}
		numStr := strings.TrimSpace(line[1:bracket])
		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		rest := strings.TrimSpace(line[bracket+1:])
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			continue
		}
		rule := UFWRule{Number: num}
		rule.To = fields[0]

		actionIdx := -1
		for i, f := range fields {
			up := strings.ToUpper(f)
			if up == "ALLOW" || up == "DENY" || up == "REJECT" || up == "LIMIT" {
				rule.Action = up
				actionIdx = i
				break
			}
		}
		if actionIdx >= 0 && actionIdx+1 < len(fields) {
			// Skip direction words (IN, OUT, FWD) that ufw appends after the action verb.
			// e.g. "ALLOW IN  Anywhere" → From = "Anywhere", not "IN Anywhere"
			dirWords := map[string]bool{"IN": true, "OUT": true, "FWD": true}
			var fromParts []string
			for _, f := range fields[actionIdx+1:] {
				if dirWords[strings.ToUpper(f)] {
					continue
				}
				fromParts = append(fromParts, f)
			}
			rule.From = strings.Join(fromParts, " ")
		}

		if idx := strings.Index(rule.To, "/"); idx >= 0 {
			rule.Protocol = rule.To[idx+1:]
			rule.To = rule.To[:idx]
		}
		rules = append(rules, rule)
	}
	if rules == nil {
		rules = []UFWRule{}
	}
	return rules, nil
}

// parseUFWRulesFiles parses /etc/ufw/user.rules as a fallback when ufw command is unavailable.
// The tuple comment lines have the format:
//
//	### tuple ### <action> <proto> <to_port> <to_addr> <from_port> <from_addr> <direction>
func parseUFWRulesFiles() ([]UFWRule, error) {
	var rules []UFWRule
	ruleNum := 1
	for _, path := range []string{"/etc/ufw/user.rules", "/etc/ufw/user6.rules"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "### tuple ###") {
				continue
			}
			// ### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in
			parts := strings.Fields(line)
			if len(parts) < 8 {
				continue
			}
			action := strings.ToUpper(parts[3])
			proto := parts[4]
			toPort := parts[5]
			fromAddr := parts[7]
			if fromAddr == "0.0.0.0/0" || fromAddr == "::/0" {
				fromAddr = "Anywhere"
			}
			rules = append(rules, UFWRule{
				Number:   ruleNum,
				To:       toPort,
				Action:   action,
				From:     fromAddr,
				Protocol: proto,
			})
			ruleNum++
		}
	}
	if rules == nil {
		return nil, fmt.Errorf("rules files not readable")
	}
	return rules, nil
}

// isUFWActive checks if UFW is enabled, first via ufw.conf then via the ufw command.
func isUFWActive() bool {
	// Fastest path: read ufw.conf (usually world-readable)
	if data, err := os.ReadFile("/etc/ufw/ufw.conf"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.EqualFold(line, "ENABLED=yes") {
				return true
			}
		}
		// ufw.conf exists but ENABLED != yes
		return false
	}
	// Fall back to running the command
	out, err := runUFW("status")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Status: active")
}

type connection struct {
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
	State      string `json:"state"`
	Protocol   string `json:"protocol"`
}

func getActiveConnections() []connection {
	out, err := exec.Command("ss", "-tnp").Output()
	if err != nil {
		return []connection{}
	}
	var conns []connection
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "State") || strings.HasPrefix(line, "Netid") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		conns = append(conns, connection{
			State:      fields[0],
			LocalAddr:  fields[3],
			RemoteAddr: fields[4],
			Protocol:   "tcp",
		})
		if len(conns) >= 50 {
			break
		}
	}
	if conns == nil {
		conns = []connection{}
	}
	return conns
}

// isValidIPOrCIDR checks if the given string is a valid IPv4/IPv6 address or CIDR notation.
func isValidIPOrCIDR(s string) bool {
	// Check for CIDR notation
	if strings.Contains(s, "/") {
		_, _, err := net.ParseCIDR(s)
		return err == nil
	}
	// Check for plain IP address
	return net.ParseIP(s) != nil
}
