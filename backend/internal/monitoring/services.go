package monitoring

import (
	"os/exec"
	"strings"
)

// ServiceStatus holds systemd service health information.
type ServiceStatus struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	ActiveState string `json:"active_state"` // active, inactive, failed, activating, deactivating
	SubState    string `json:"sub_state"`    // running, dead, exited, failed, etc.
	IsRunning   bool   `json:"running"`
}

// CheckServices queries systemctl for each service in the list.
func CheckServices(services []string) []ServiceStatus {
	result := make([]ServiceStatus, 0, len(services))
	for _, svc := range services {
		st := checkOne(svc)
		result = append(result, st)
	}
	return result
}

func checkOne(name string) ServiceStatus {
	svc := ServiceStatus{
		Name:        name,
		Label:       serviceLabel(name),
		ActiveState: "inactive",
		SubState:    "dead",
	}

	out, err := exec.Command("systemctl", "show", "--no-pager",
		"--property=ActiveState,SubState", name).Output()
	if err != nil {
		return svc
	}

	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "ActiveState":
			svc.ActiveState = strings.TrimSpace(parts[1])
		case "SubState":
			svc.SubState = strings.TrimSpace(parts[1])
		}
	}

	svc.IsRunning = svc.ActiveState == "active" && svc.SubState == "running"
	// UFW is a oneshot service that shows "active:exited" when working
	if name == "ufw" && svc.ActiveState == "active" && svc.SubState == "exited" {
		svc.IsRunning = true
	}
	return svc
}

func serviceLabel(name string) string {
	labels := map[string]string{
		"ufw":                 "UFW",
		"fail2ban":            "fail2ban",
		"crowdsec":            "CrowdSec",
		"psad":                "psad",
		"clamav-daemon":       "ClamAV",
		"auditd":              "auditd",
		"apparmor":            "AppArmor",
		"docker":              "Docker",
		"netdata":             "Netdata",
		"unattended-upgrades": "Auto-Update",
		"aide":                "AIDE",
		"rkhunter":            "rkhunter",
		"nginx":               "nginx",
		"sshd":                "sshd",
	}
	if l, ok := labels[name]; ok {
		return l
	}
	return name
}
