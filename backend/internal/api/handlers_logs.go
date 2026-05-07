package api

import (
	"bufio"
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// logSources maps source names to candidate file paths on the filesystem.
// The first path that exists and is readable wins. Journalctl is the fallback.
var logSources = map[string][]string{
	"auth":        {"/var/log/auth.log", "/var/log/secure"},
	"syslog":      {"/var/log/syslog", "/var/log/messages"},
	"kern":        {"/var/log/kern.log", "/var/log/kernel"},
	"nginx":       {"/var/log/nginx/access.log"},
	"nginx-error": {"/var/log/nginx/error.log"},
	"docker":      {"/var/log/docker.log", "/var/log/upstart/docker.log"},
	"crowdsec":    {"/var/log/crowdsec/crowdsec.log"},
	"fail2ban":    {"/var/log/fail2ban.log"},
	"journal":     {},
}

// journaldUnit maps source names to systemd unit names for journalctl fallback.
var journaldUnit = map[string]string{
	"auth":     "ssh",
	"docker":   "docker",
	"crowdsec": "crowdsec",
	"fail2ban": "fail2ban",
	"journal":  "",
}

func (h *Handlers) GetLogs(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "syslog"
	}
	linesParam, _ := strconv.Atoi(r.URL.Query().Get("lines"))
	if linesParam <= 0 || linesParam > 5000 {
		linesParam = 200
	}

	var logLines []string

	// Try filesystem paths first
	if paths, ok := logSources[source]; ok {
		for _, path := range paths {
			if lines, err := tailFile(path, linesParam); err == nil {
				logLines = lines
				break
			}
		}
	}

	// Fall back to journalctl
	if len(logLines) == 0 {
		logLines = journalFallback(source, linesParam)
	}

	if logLines == nil {
		logLines = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source": source,
		"lines":  logLines,
	})
}

// tailFile reads the last n lines of a file efficiently.
func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read entire file into ring buffer of n lines
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	ring := make([]string, n)
	idx := 0
	total := 0
	for scanner.Scan() {
		ring[idx%n] = scanner.Text()
		idx++
		total++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if total == 0 {
		return []string{}, nil
	}

	start := idx % n
	result := make([]string, 0, n)
	for i := 0; i < n && i < total; i++ {
		line := ring[(start+i)%n]
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// journalFallback reads logs from journalctl.
func journalFallback(source string, n int) []string {
	args := []string{"--no-pager", "-n", strconv.Itoa(n), "--output=short-iso"}
	if unit, ok := journaldUnit[source]; ok && unit != "" {
		args = append(args, "-u", unit)
	}
	out, err := exec.Command("journalctl", args...).Output()
	if err != nil {
		return []string{}
	}
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "--") {
			lines = append(lines, line)
		}
	}
	return lines
}
