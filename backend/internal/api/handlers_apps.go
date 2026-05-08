package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	chi "github.com/go-chi/chi/v5"

	"github.com/ehsanR91/sentinelcore/internal/db"
)

// ── App catalog ───────────────────────────────────────────────────────────────

type appCatalogItem struct {
	Label       string
	Description string
	Category    string // cli | runtime | web | database | build | devtool | shell
	Homepage    string
	Binary      string   // primary binary for install-check / version query
	VersionArgs []string // args passed to Binary to get version string
	VersionRx   string   // regex to extract semver from version output (capture group 1)
	// Per-distro package lists
	AptPkgs       []string
	AptCheckPkg   string // package name for apt-cache policy / dpkg-query
	DnfPkgs       []string
	DnfCheckPkg   string
	PacmanPkgs    []string
	ZypperPkgs    []string
	UninstallPkgs []string // subset to remove on uninstall
	// How to install (determines the install goroutine logic)
	InstallMethod string // pkg | script | binary | rustup
	// For binary / API-based version checks (Go, Rust)
	LatestAPI string
	LatestRx  string
}

var appCatalog = map[string]appCatalogItem{
	"essential-cli": {
		Label:       "Essential CLI",
		Description: "Core dev tools: git, curl, wget, jq, rsync, tmux, vim, htop, ncdu, fzf, ripgrep, bat, tree, unzip, and many more",
		Category:    "cli",
		Homepage:    "https://git-scm.com",
		Binary:      "git",
		VersionArgs: []string{"--version"},
		VersionRx:   `git version (.+)`,
		AptPkgs: []string{
			"git", "curl", "wget", "jq", "rsync", "tmux", "vim", "htop", "ncdu",
			"fzf", "ripgrep", "bat", "fd-find", "silversearcher-ag",
			"tree", "unzip", "zip", "pv",
			"dnsutils", "netcat-openbsd", "traceroute", "moreutils",
			"software-properties-common", "apt-transport-https",
		},
		AptCheckPkg: "git",
		DnfPkgs: []string{
			"git", "curl", "wget", "jq", "rsync", "tmux", "vim", "htop", "ncdu",
			"fzf", "ripgrep", "bat", "the_silver_searcher",
			"tree", "unzip", "zip", "bind-utils", "nmap-ncat", "traceroute", "moreutils",
		},
		DnfCheckPkg:   "git",
		PacmanPkgs:    []string{"git", "curl", "wget", "jq", "rsync", "tmux", "vim", "htop", "ncdu", "fzf", "ripgrep", "bat", "fd", "the_silver_searcher", "tree", "unzip", "zip", "traceroute"},
		ZypperPkgs:    []string{"git", "curl", "wget", "jq", "rsync", "tmux", "vim", "htop", "ncdu", "fzf", "ripgrep", "bat", "tree", "unzip", "zip", "traceroute"},
		InstallMethod: "pkg",
		UninstallPkgs: []string{"git", "curl", "wget", "jq", "rsync", "tmux", "vim", "htop", "ncdu", "fzf", "ripgrep", "bat", "tree", "unzip", "zip"},
	},
	"docker": {
		Label:         "Docker CE",
		Description:   "Docker container runtime with buildx and compose plugins. Installed from the official Docker repository.",
		Category:      "runtime",
		Homepage:      "https://docs.docker.com",
		Binary:        "docker",
		VersionArgs:   []string{"--version"},
		VersionRx:     `Docker version ([0-9.]+)`,
		AptPkgs:       []string{"docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"},
		AptCheckPkg:   "docker-ce",
		DnfPkgs:       []string{"docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"},
		DnfCheckPkg:   "docker-ce",
		PacmanPkgs:    []string{"docker", "docker-buildx", "docker-compose"},
		ZypperPkgs:    []string{"docker", "docker-compose"},
		InstallMethod: "script",
		UninstallPkgs: []string{"docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"},
	},
	"nodejs": {
		Label:         "Node.js LTS",
		Description:   "Node.js JavaScript runtime (current LTS) with npm and npx. Installed via the official NodeSource repository.",
		Category:      "runtime",
		Homepage:      "https://nodejs.org",
		Binary:        "node",
		VersionArgs:   []string{"--version"},
		VersionRx:     `v(.+)`,
		AptPkgs:       []string{"nodejs"},
		AptCheckPkg:   "nodejs",
		DnfPkgs:       []string{"nodejs", "npm"},
		DnfCheckPkg:   "nodejs",
		PacmanPkgs:    []string{"nodejs", "npm"},
		ZypperPkgs:    []string{"nodejs20", "npm20"},
		InstallMethod: "script",
		UninstallPkgs: []string{"nodejs", "npm"},
	},
	"python3": {
		Label:         "Python 3",
		Description:   "Python 3 with pip, venv, dev headers, and pipx for isolated tool installs",
		Category:      "runtime",
		Homepage:      "https://python.org",
		Binary:        "python3",
		VersionArgs:   []string{"--version"},
		VersionRx:     `Python (.+)`,
		AptPkgs:       []string{"python3", "python3-pip", "python3-venv", "python3-dev", "pipx"},
		AptCheckPkg:   "python3",
		DnfPkgs:       []string{"python3", "python3-pip", "python3-devel"},
		DnfCheckPkg:   "python3",
		PacmanPkgs:    []string{"python", "python-pip", "python-pipx"},
		ZypperPkgs:    []string{"python3", "python3-pip", "python3-devel"},
		InstallMethod: "pkg",
		UninstallPkgs: []string{"python3-pip", "python3-venv", "python3-dev", "pipx"},
	},
	"golang": {
		Label:         "Go",
		Description:   "Go programming language — latest stable binary from go.dev, installed to /usr/local/go with PATH configured in /etc/profile.d/golang.sh",
		Category:      "runtime",
		Homepage:      "https://go.dev",
		Binary:        "go",
		VersionArgs:   []string{"version"},
		VersionRx:     `go version go(\S+)`,
		InstallMethod: "binary",
		LatestAPI:     "https://go.dev/VERSION?m=text",
		LatestRx:      `^go(.+)`,
	},
	"nginx": {
		Label:         "nginx + Certbot",
		Description:   "Nginx high-performance web server with Certbot for automated Let's Encrypt TLS certificates",
		Category:      "web",
		Homepage:      "https://nginx.org",
		Binary:        "nginx",
		VersionArgs:   []string{"-v"},
		VersionRx:     `nginx/(.+)`,
		AptPkgs:       []string{"nginx", "certbot", "python3-certbot-nginx"},
		AptCheckPkg:   "nginx",
		DnfPkgs:       []string{"nginx", "certbot", "python3-certbot-nginx"},
		DnfCheckPkg:   "nginx",
		PacmanPkgs:    []string{"nginx", "certbot", "certbot-nginx"},
		ZypperPkgs:    []string{"nginx", "python3-certbot-nginx"},
		InstallMethod: "pkg",
		UninstallPkgs: []string{"nginx", "certbot", "python3-certbot-nginx"},
	},
	"db-clients": {
		Label:         "Database Clients",
		Description:   "PostgreSQL client (psql), Redis CLI, mycli (MySQL), and pgcli (Postgres) for database management",
		Category:      "database",
		Homepage:      "https://www.postgresql.org",
		Binary:        "psql",
		VersionArgs:   []string{"--version"},
		VersionRx:     `psql \(PostgreSQL\) (.+)`,
		AptPkgs:       []string{"postgresql-client", "redis-tools"},
		AptCheckPkg:   "postgresql-client",
		DnfPkgs:       []string{"postgresql", "redis"},
		DnfCheckPkg:   "postgresql",
		PacmanPkgs:    []string{"postgresql-libs", "redis"},
		ZypperPkgs:    []string{"postgresql", "redis"},
		InstallMethod: "pkg",
		UninstallPkgs: []string{"postgresql-client", "redis-tools"},
	},
	"build-tools": {
		Label:         "Build Tools",
		Description:   "Compilers and build infrastructure: gcc, g++, make, cmake, pkg-config, libssl-dev, autoconf, automake, libtool",
		Category:      "build",
		Homepage:      "https://gcc.gnu.org",
		Binary:        "gcc",
		VersionArgs:   []string{"--version"},
		VersionRx:     `gcc \([^)]+\) (\S+)`,
		AptPkgs:       []string{"build-essential", "cmake", "pkg-config", "libssl-dev", "libffi-dev", "autoconf", "automake", "libtool"},
		AptCheckPkg:   "build-essential",
		DnfPkgs:       []string{"gcc", "gcc-c++", "make", "cmake", "pkg-config", "openssl-devel", "libffi-devel", "autoconf", "automake", "libtool"},
		DnfCheckPkg:   "gcc",
		PacmanPkgs:    []string{"base-devel", "cmake", "openssl"},
		ZypperPkgs:    []string{"gcc", "gcc-c++", "cmake", "pkg-config", "libopenssl-devel", "autoconf", "automake", "libtool"},
		InstallMethod: "pkg",
		UninstallPkgs: []string{"build-essential", "cmake", "autoconf", "automake", "libtool"},
	},
	"github-cli": {
		Label:         "GitHub CLI",
		Description:   "Official GitHub command-line interface (gh) for managing issues, pull requests, and repositories",
		Category:      "devtool",
		Homepage:      "https://cli.github.com",
		Binary:        "gh",
		VersionArgs:   []string{"--version"},
		VersionRx:     `gh version (\S+)`,
		AptPkgs:       []string{"gh"},
		AptCheckPkg:   "gh",
		DnfPkgs:       []string{"gh"},
		DnfCheckPkg:   "gh",
		PacmanPkgs:    []string{"github-cli"},
		ZypperPkgs:    []string{"gh"},
		InstallMethod: "script",
		UninstallPkgs: []string{"gh"},
	},
	"rust": {
		Label:         "Rust",
		Description:   "Rust systems programming language installed via rustup for the admin user. Includes cargo, rustfmt, and clippy.",
		Category:      "runtime",
		Homepage:      "https://www.rust-lang.org",
		Binary:        "rustc",
		VersionArgs:   []string{"--version"},
		VersionRx:     `rustc (\S+)`,
		InstallMethod: "rustup",
		LatestAPI:     "https://static.rust-lang.org/dist/channel-rust-stable.toml",
		LatestRx:      `\[pkg\.rustc\]\nversion = "(.+?)"`,
	},
	"zsh": {
		Label:         "Zsh + Oh-My-Zsh",
		Description:   "Z shell with Oh-My-Zsh framework, zsh-autosuggestions, and zsh-syntax-highlighting plugins installed for the admin user",
		Category:      "shell",
		Homepage:      "https://ohmyz.sh",
		Binary:        "zsh",
		VersionArgs:   []string{"--version"},
		VersionRx:     `zsh (\S+)`,
		AptPkgs:       []string{"zsh"},
		AptCheckPkg:   "zsh",
		DnfPkgs:       []string{"zsh"},
		DnfCheckPkg:   "zsh",
		PacmanPkgs:    []string{"zsh"},
		ZypperPkgs:    []string{"zsh"},
		InstallMethod: "script",
		UninstallPkgs: []string{"zsh"},
	},
}

// ── Response struct ───────────────────────────────────────────────────────────

type managedApp struct {
	Name          string `json:"name"`
	Label         string `json:"label"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Homepage      string `json:"homepage"`
	Binary        string `json:"binary"`
	InstallMethod string `json:"install_method"`
	Installed     bool   `json:"installed"`
	Status        string `json:"status"`      // not_installed | installing | updating | uninstalling | installed | failed
	Version       string `json:"version"`     // installed version (empty if not installed)
	NewVersion    string `json:"new_version"` // latest available (empty if unknown or up-to-date)
	UpdateAvail   bool   `json:"update_avail"`
}

// ── Background operation state ────────────────────────────────────────────────

var (
	appOpMu      sync.Mutex
	appOpRunning bool
	appOpName    string // which app is being operated on
	appOpKind    string // "install" | "update" | "uninstall"
	appOpLogs    []string
	appOpDone    bool
	appOpErr     string
)

// ── Version cache (10-minute TTL so page loads stay fast) ────────────────────

type appVerEntry struct {
	ver string
	at  time.Time
}

var (
	appVerCacheMu sync.Mutex
	appVerCache   = map[string]appVerEntry{}
)

const appVerCacheTTL = 10 * time.Minute

// ── Handlers ─────────────────────────────────────────────────────────────────

// GetApps returns the full app catalog with install status and version info.
// Available-version checks are done in parallel with a shared 8s deadline.
func (h *Handlers) GetApps(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	appOpMu.Lock()
	opRunning, opName, opKind := appOpRunning, appOpName, appOpKind
	appOpMu.Unlock()

	keys := make([]string, 0, len(appCatalog))
	for name := range appCatalog {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	// Fetch available versions for all apps in parallel.
	type verResult struct {
		name string
		ver  string
	}
	vCh := make(chan verResult, len(keys))
	for _, name := range keys {
		go func(n string, m appCatalogItem) {
			vCh <- verResult{n, appAvailableVersionCached(ctx, n, m)}
		}(name, appCatalog[name])
	}
	avail := make(map[string]string, len(keys))
	for range keys {
		r2 := <-vCh
		avail[r2.name] = r2.ver
	}

	out := make([]managedApp, 0, len(keys))
	for _, name := range keys {
		meta := appCatalog[name]
		installed := appInstalled(meta)

		version := ""
		if installed {
			version = appInstalledVersion(meta)
		}

		newVer := avail[name]
		updateAvail := installed && newVer != "" && !versionEqual(version, newVer)

		status := "not_installed"
		if installed {
			status = "installed"
		}
		if opRunning && opName == name {
			switch opKind {
			case "install":
				status = "installing"
			case "update":
				status = "updating"
			case "uninstall":
				status = "uninstalling"
			}
		}

		out = append(out, managedApp{
			Name:          name,
			Label:         meta.Label,
			Description:   meta.Description,
			Category:      meta.Category,
			Homepage:      meta.Homepage,
			Binary:        meta.Binary,
			InstallMethod: meta.InstallMethod,
			Installed:     installed,
			Status:        status,
			Version:       version,
			NewVersion:    newVer,
			UpdateAvail:   updateAvail,
		})
	}

	writeJSON(w, http.StatusOK, out)
}

// InstallApp starts a background installation of the named app.
func (h *Handlers) InstallApp(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	meta, ok := appCatalog[name]
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown app")
		return
	}
	if appInstalled(meta) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "already_installed"})
		return
	}
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	appOpMu.Lock()
	if appOpRunning {
		appOpMu.Unlock()
		writeError(w, http.StatusConflict, "another app operation is already in progress")
		return
	}
	appOpRunning, appOpName, appOpKind = true, name, "install"
	appOpLogs, appOpDone, appOpErr = []string{}, false, ""
	appOpMu.Unlock()

	go func() {
		defer func() {
			appOpMu.Lock()
			appOpRunning = false
			appOpMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		appLog(fmt.Sprintf("=== Installing %s ===", meta.Label))
		if err := doInstallApp(ctx, name, meta); err != nil {
			appOpMu.Lock()
			appOpErr = err.Error()
			appOpDone = true
			appOpMu.Unlock()
			db.InsertAlert("app", "error", "system", fmt.Sprintf("App install failed: %s — %v", meta.Label, err), "", "system")
			return
		}
		appLog(fmt.Sprintf("=== %s installed successfully ===", meta.Label))
		invalidateAppVerCache(name)
		appOpMu.Lock()
		appOpDone = true
		appOpMu.Unlock()
		db.InsertAlert("app", "info", "system", fmt.Sprintf("App installed: %s", meta.Label), "", "system")
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "installing"})
}

// UpdateApp starts a background upgrade of an already-installed app.
func (h *Handlers) UpdateApp(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	meta, ok := appCatalog[name]
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown app")
		return
	}
	if !appInstalled(meta) {
		writeError(w, http.StatusBadRequest, "app is not installed")
		return
	}
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	appOpMu.Lock()
	if appOpRunning {
		appOpMu.Unlock()
		writeError(w, http.StatusConflict, "another app operation is already in progress")
		return
	}
	appOpRunning, appOpName, appOpKind = true, name, "update"
	appOpLogs, appOpDone, appOpErr = []string{}, false, ""
	appOpMu.Unlock()

	go func() {
		defer func() {
			appOpMu.Lock()
			appOpRunning = false
			appOpMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		appLog(fmt.Sprintf("=== Updating %s ===", meta.Label))
		if err := doUpdateApp(ctx, name, meta); err != nil {
			appOpMu.Lock()
			appOpErr = err.Error()
			appOpDone = true
			appOpMu.Unlock()
			return
		}
		appLog(fmt.Sprintf("=== %s updated successfully ===", meta.Label))
		invalidateAppVerCache(name)
		appOpMu.Lock()
		appOpDone = true
		appOpMu.Unlock()
		db.InsertAlert("app", "info", "system", fmt.Sprintf("App updated: %s", meta.Label), "", "system")
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "updating"})
}

// UninstallApp starts a background removal of an installed app.
func (h *Handlers) UninstallApp(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	meta, ok := appCatalog[name]
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown app")
		return
	}
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	appOpMu.Lock()
	if appOpRunning {
		appOpMu.Unlock()
		writeError(w, http.StatusConflict, "another app operation is already in progress")
		return
	}
	appOpRunning, appOpName, appOpKind = true, name, "uninstall"
	appOpLogs, appOpDone, appOpErr = []string{}, false, ""
	appOpMu.Unlock()

	go func() {
		defer func() {
			appOpMu.Lock()
			appOpRunning = false
			appOpMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		appLog(fmt.Sprintf("=== Uninstalling %s ===", meta.Label))
		if err := doUninstallApp(ctx, name, meta); err != nil {
			appOpMu.Lock()
			appOpErr = err.Error()
			appOpDone = true
			appOpMu.Unlock()
			return
		}
		appLog(fmt.Sprintf("=== %s removed ===", meta.Label))
		invalidateAppVerCache(name)
		appOpMu.Lock()
		appOpDone = true
		appOpMu.Unlock()
		db.InsertAlert("app", "info", "system", fmt.Sprintf("App removed: %s", meta.Label), "", "system")
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "uninstalling"})
}

// GetAppOpLogs returns the live log stream and done/error state for any running app operation.
func (h *Handlers) GetAppOpLogs(w http.ResponseWriter, r *http.Request) {
	appOpMu.Lock()
	defer appOpMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"logs":    appOpLogs,
		"done":    appOpDone,
		"error":   appOpErr,
		"app":     appOpName,
		"kind":    appOpKind,
		"running": appOpRunning,
	})
}

// ── Install logic ─────────────────────────────────────────────────────────────

func doInstallApp(ctx context.Context, name string, meta appCatalogItem) error {
	switch name {
	case "docker":
		return installDocker(ctx, meta)
	case "nodejs":
		return installNodeJS(ctx, meta)
	case "golang":
		return installGo(ctx)
	case "github-cli":
		return installGitHubCLI(ctx, meta)
	case "rust":
		return installRust(ctx)
	case "zsh":
		return installZsh(ctx, meta)
	default:
		return installViaPkg(ctx, meta)
	}
}

func doUpdateApp(ctx context.Context, name string, meta appCatalogItem) error {
	switch name {
	case "golang":
		return installGo(ctx) // re-download latest
	case "rust":
		out, err := runPrivilegedShell(ctx, "rustup update stable 2>&1 || true")
		appLogMulti(out)
		return err
	case "zsh":
		// update Oh-My-Zsh for detected user
		user := findAdminUser()
		if user != "" {
			out, err := runPrivilegedShell(ctx, fmt.Sprintf(
				`sudo -u '%s' bash -c 'cd ~/.oh-my-zsh && git pull 2>&1' || true`, user))
			appLogMulti(out)
			return err
		}
		return nil
	default:
		return upgradeViaPkg(ctx, meta)
	}
}

func doUninstallApp(ctx context.Context, name string, meta appCatalogItem) error {
	switch name {
	case "golang":
		out, err := runPrivilegedShell(ctx, "rm -rf /usr/local/go && rm -f /etc/profile.d/golang.sh && echo 'Go removed'")
		appLogMulti(out)
		return err
	case "rust":
		user := findAdminUser()
		if user != "" {
			out, err := runPrivilegedShell(ctx, fmt.Sprintf(
				`sudo -u '%s' bash -c '~/.cargo/bin/rustup self uninstall -y 2>&1' || echo 'rustup not found for %s'`, user, user))
			appLogMulti(out)
			return err
		}
		return nil
	case "zsh":
		user := findAdminUser()
		if user != "" {
			// Uninstall Oh-My-Zsh first, then remove zsh
			out, _ := runPrivilegedShell(ctx, fmt.Sprintf(
				`sudo -u '%s' bash -c 'unset ZSH; ~/.oh-my-zsh/tools/uninstall.sh 2>&1' || true`, user))
			appLogMulti(out)
		}
		return removeViaPkg(ctx, meta)
	default:
		return removeViaPkg(ctx, meta)
	}
}

// ── Per-app install implementations ──────────────────────────────────────────

func installViaPkg(ctx context.Context, meta appCatalogItem) error {
	pkgs := appPkgsForDistro(meta)
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages defined for distro '%s'", pkgFamily)
	}
	// Fix any repo key issues before installing
	if fixed := fixMissingRepoKeys(ctx); len(fixed) > 0 {
		appLog(fmt.Sprintf("Fixed GPG keys: %s", strings.Join(fixed, ", ")))
	}
	if pkgFamily == "apt" {
		out, err := runPrivilegedEnv(ctx, "apt-get", append([]string{"-y", "update"}, "2>&1")...)
		appLogMulti(out)
		if err != nil {
			appLog("apt-get update had errors, continuing...")
		}
		out, err = runPrivilegedEnv(ctx, "apt-get", append([]string{"-y", "install"}, pkgs...)...)
		appLogMulti(out)
		return err
	}
	args, pm := pkgInstallArgs(pkgs)
	out, err := runPrivilegedEnv(ctx, pm, args...)
	appLogMulti(out)
	return err
}

func upgradeViaPkg(ctx context.Context, meta appCatalogItem) error {
	checkPkg := appCheckPkgForDistro(meta)
	if checkPkg == "" {
		return fmt.Errorf("no update method for '%s' on distro '%s'", meta.Label, pkgFamily)
	}
	var out string
	var err error
	switch pkgFamily {
	case "apt":
		out, err = runPrivilegedEnv(ctx, "apt-get", "-y", "install", "--only-upgrade", checkPkg)
	case "dnf", "yum":
		out, err = runPrivilegedEnv(ctx, pkgFamily, "-y", "upgrade", checkPkg)
	case "pacman":
		out, err = runPrivilegedEnv(ctx, "pacman", "-Syu", "--noconfirm", checkPkg)
	case "zypper":
		out, err = runPrivilegedEnv(ctx, "zypper", "-n", "update", checkPkg)
	}
	appLogMulti(out)
	return err
}

func removeViaPkg(ctx context.Context, meta appCatalogItem) error {
	pkgs := meta.UninstallPkgs
	if len(pkgs) == 0 {
		pkgs = appPkgsForDistro(meta)
	}
	if len(pkgs) == 0 {
		return nil
	}
	var out string
	var err error
	switch pkgFamily {
	case "apt":
		out, err = runPrivilegedEnv(ctx, "apt-get", append([]string{"-y", "remove", "--purge"}, pkgs...)...)
		appLogMulti(out)
		if err == nil {
			out2, _ := runPrivilegedEnv(ctx, "apt-get", "-y", "autoremove")
			appLogMulti(out2)
		}
	case "dnf", "yum":
		out, err = runPrivilegedEnv(ctx, pkgFamily, append([]string{"-y", "remove"}, pkgs...)...)
		appLogMulti(out)
	case "pacman":
		out, err = runPrivilegedEnv(ctx, "pacman", append([]string{"-Rs", "--noconfirm"}, pkgs...)...)
		appLogMulti(out)
	case "zypper":
		out, err = runPrivilegedEnv(ctx, "zypper", append([]string{"-n", "remove"}, pkgs...)...)
		appLogMulti(out)
	}
	return err
}

func installDocker(ctx context.Context, meta appCatalogItem) error {
	switch pkgFamily {
	case "apt":
		script := `set -euo pipefail
apt-get install -y -q ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
ARCH=$(dpkg --print-architecture)
CODENAME=$(. /etc/os-release && echo "${VERSION_CODENAME:-$(lsb_release -cs 2>/dev/null)}")
echo "deb [arch=${ARCH} signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu ${CODENAME} stable" \
  > /etc/apt/sources.list.d/docker.list
apt-get update -qq
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable docker
systemctl start docker
docker --version`
		out, err := runPrivilegedShell(ctx, script)
		appLogMulti(out)
		return err
	case "dnf", "yum":
		script := `set -euo pipefail
` + pkgFamily + ` install -y -q yum-utils
` + pkgFamily + `-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
` + pkgFamily + ` install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable docker
systemctl start docker
docker --version`
		out, err := runPrivilegedShell(ctx, script)
		appLogMulti(out)
		return err
	case "pacman":
		out, err := runPrivilegedEnv(ctx, "pacman", "-Sy", "--noconfirm", "docker", "docker-buildx", "docker-compose")
		appLogMulti(out)
		if err == nil {
			out2, _ := runPrivileged(ctx, "systemctl", "enable", "--now", "docker")
			appLogMulti(out2)
		}
		return err
	default:
		return installViaPkg(ctx, meta)
	}
}

func installNodeJS(ctx context.Context, meta appCatalogItem) error {
	switch pkgFamily {
	case "apt":
		script := `set -euo pipefail
curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
apt-get install -y nodejs
node --version
npm --version`
		out, err := runPrivilegedShell(ctx, script)
		appLogMulti(out)
		return err
	case "dnf", "yum":
		script := `set -euo pipefail
curl -fsSL https://rpm.nodesource.com/setup_lts.x | bash -
` + pkgFamily + ` install -y nodejs
node --version`
		out, err := runPrivilegedShell(ctx, script)
		appLogMulti(out)
		return err
	default:
		return installViaPkg(ctx, meta)
	}
}

func installGo(ctx context.Context) error {
	goArch := "amd64"
	if runtime.GOARCH == "arm64" {
		goArch = "arm64"
	}
	script := fmt.Sprintf(`set -euo pipefail
GO_VER=$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -1)
echo "Latest Go: ${GO_VER}"
wget -qO /tmp/go.tar.gz "https://go.dev/dl/${GO_VER}.linux-%s.tar.gz"

# Try to remove existing Go installation with error handling
if [ -d /usr/local/go ]; then
    echo "Removing existing Go installation..."
    # First try normal removal
    rm -rf /usr/local/go 2>/dev/null || {
        echo "Normal removal failed, trying with force..."
        # If that fails, try with more forceful approach
        find /usr/local/go -type f -delete 2>/dev/null || true
        find /usr/local/go -type d -delete 2>/dev/null || true
        rmdir /usr/local/go 2>/dev/null || {
            echo "Warning: Could not completely remove /usr/local/go"
            echo "Attempting to move it aside..."
            mv /usr/local/go /usr/local/go.backup.$(date +%%s) 2>/dev/null || true
        }
    }
fi

# Extract new Go installation
echo "Extracting Go to /usr/local..."
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz

# Set up environment
echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/golang.sh
chmod 644 /etc/profile.d/golang.sh

# Verify installation
if [ -f /usr/local/go/bin/go ]; then
    /usr/local/go/bin/go version
    echo "Go installation completed successfully"
else
    echo "ERROR: Go binary not found after installation"
    exit 1
fi`, goArch)
	out, err := runPrivilegedShell(ctx, script)
	appLogMulti(out)
	return err
}

func installGitHubCLI(ctx context.Context, meta appCatalogItem) error {
	switch pkgFamily {
	case "apt":
		script := `set -euo pipefail
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
  | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
ARCH=$(dpkg --print-architecture)
echo "deb [arch=${ARCH} signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] \
  https://cli.github.com/packages stable main" \
  > /etc/apt/sources.list.d/github-cli.list
apt-get update -qq
apt-get install -y gh
gh --version`
		out, err := runPrivilegedShell(ctx, script)
		appLogMulti(out)
		return err
	case "dnf", "yum":
		script := `set -euo pipefail
` + pkgFamily + ` install -y 'dnf-command(config-manager)' 2>/dev/null || true
` + pkgFamily + `-config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo 2>/dev/null || \
  ` + pkgFamily + ` config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo
` + pkgFamily + ` install -y gh
gh --version`
		out, err := runPrivilegedShell(ctx, script)
		appLogMulti(out)
		return err
	case "pacman":
		out, err := runPrivilegedEnv(ctx, "pacman", "-Sy", "--noconfirm", "github-cli")
		appLogMulti(out)
		return err
	default:
		return installViaPkg(ctx, meta)
	}
}

func installRust(ctx context.Context) error {
	user := findAdminUser()
	var script string
	if user != "" {
		script = fmt.Sprintf(`set -euo pipefail
echo "Installing Rust for user: %s"
sudo -u '%s' bash -c 'curl -fsSL https://sh.rustup.rs | sh -s -- -y --no-modify-path 2>&1'
BASHRC="/home/%s/.bashrc"
grep -q 'cargo/bin' "$BASHRC" 2>/dev/null || \
  echo 'export PATH="$HOME/.cargo/bin:$PATH"' >> "$BASHRC"
sudo -u '%s' ~/.cargo/bin/rustc --version 2>/dev/null || true`, user, user, user, user)
	} else {
		script = `set -euo pipefail
curl -fsSL https://sh.rustup.rs | sh -s -- -y --no-modify-path 2>&1
source "$HOME/.cargo/env"
rustc --version`
	}
	out, err := runPrivilegedShell(ctx, script)
	appLogMulti(out)
	return err
}

func installZsh(ctx context.Context, meta appCatalogItem) error {
	// Install zsh package first
	if err := installViaPkg(ctx, meta); err != nil {
		return err
	}
	user := findAdminUser()
	if user == "" {
		appLog("No admin user detected; skipping Oh-My-Zsh install")
		return nil
	}
	script := fmt.Sprintf(`set -euo pipefail
ZSH_USER='%s'
HOME_DIR="/home/${ZSH_USER}"
# Install Oh-My-Zsh if not present
if [[ ! -d "${HOME_DIR}/.oh-my-zsh" ]]; then
  echo "Installing Oh-My-Zsh for ${ZSH_USER}..."
  sudo -u "${ZSH_USER}" bash -c \
    'RUNZSH=no CHSH=no sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" 2>&1'
fi
PLUGINS="${HOME_DIR}/.oh-my-zsh/custom/plugins"
sudo -u "${ZSH_USER}" mkdir -p "${PLUGINS}"
# Clone autosuggestions
if [[ ! -d "${PLUGINS}/zsh-autosuggestions" ]]; then
  sudo -u "${ZSH_USER}" git clone --quiet \
    https://github.com/zsh-users/zsh-autosuggestions \
    "${PLUGINS}/zsh-autosuggestions"
fi
# Clone syntax highlighting
if [[ ! -d "${PLUGINS}/zsh-syntax-highlighting" ]]; then
  sudo -u "${ZSH_USER}" git clone --quiet \
    https://github.com/zsh-users/zsh-syntax-highlighting \
    "${PLUGINS}/zsh-syntax-highlighting"
fi
# Enable plugins in .zshrc
ZSHRC="${HOME_DIR}/.zshrc"
if [[ -f "${ZSHRC}" ]]; then
  grep -q 'zsh-autosuggestions' "${ZSHRC}" || \
    sed -i 's/^plugins=(\(.*\))/plugins=(\1 zsh-autosuggestions zsh-syntax-highlighting)/' "${ZSHRC}" 2>/dev/null || true
fi
chsh -s /bin/zsh "${ZSH_USER}" 2>/dev/null || true
echo "Zsh + Oh-My-Zsh installed for ${ZSH_USER}"
zsh --version`, user)

	out, err := runPrivilegedShell(ctx, script)
	appLogMulti(out)
	return err
}

// ── Version detection ─────────────────────────────────────────────────────────

// appInstalled checks whether an app's primary binary exists or its check-package is installed.
func appInstalled(meta appCatalogItem) bool {
	if meta.Binary != "" {
		// Normal PATH lookup
		if _, err := exec.LookPath(meta.Binary); err == nil {
			return true
		}
		// Go is installed outside PATH for daemon processes
		if meta.Binary == "go" {
			if _, err := os.Stat("/usr/local/go/bin/go"); err == nil {
				return true
			}
		}
		// Rust may be in user home dirs
		if meta.Binary == "rustc" {
			for _, dir := range rustCandidateDirs() {
				if _, err := os.Stat(dir + "/.cargo/bin/rustc"); err == nil {
					return true
				}
			}
		}
		// Oh-My-Zsh: check directory presence for any user
		if meta.Binary == "zsh" {
			for _, dir := range rustCandidateDirs() {
				if _, err := os.Stat(dir + "/.oh-my-zsh"); err == nil {
					return true
				}
			}
			if _, err := exec.LookPath("zsh"); err == nil {
				return true
			}
		}
	}
	// Fall back to package manager
	checkPkg := appCheckPkgForDistro(meta)
	if checkPkg != "" {
		return packageInstalled(checkPkg)
	}
	return false
}

// appInstalledVersion runs the app binary and extracts the version number.
func appInstalledVersion(meta appCatalogItem) string {
	bin := meta.Binary
	// Resolve absolute paths for binaries not always in daemon PATH
	if bin == "go" {
		if _, err := exec.LookPath("go"); err != nil {
			bin = "/usr/local/go/bin/go"
		}
	}
	if bin == "rustc" {
		for _, dir := range rustCandidateDirs() {
			p := dir + "/.cargo/bin/rustc"
			if _, err := os.Stat(p); err == nil {
				bin = p
				break
			}
		}
	}

	out, err := exec.Command(bin, meta.VersionArgs...).CombinedOutput()
	if err != nil && len(out) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(out))
	if meta.VersionRx == "" {
		return strings.Split(s, "\n")[0]
	}
	rx := regexp.MustCompile(meta.VersionRx)
	if m := rx.FindStringSubmatch(s); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return strings.Split(s, "\n")[0]
}

// appAvailableVersionCached returns the latest available version, using a 10-min in-memory cache.
func appAvailableVersionCached(ctx context.Context, name string, meta appCatalogItem) string {
	appVerCacheMu.Lock()
	if e, ok := appVerCache[name]; ok && time.Since(e.at) < appVerCacheTTL {
		appVerCacheMu.Unlock()
		return e.ver
	}
	appVerCacheMu.Unlock()

	v := fetchAvailableVersion(ctx, name, meta)

	appVerCacheMu.Lock()
	appVerCache[name] = appVerEntry{ver: v, at: time.Now()}
	appVerCacheMu.Unlock()
	return v
}

func invalidateAppVerCache(name string) {
	appVerCacheMu.Lock()
	delete(appVerCache, name)
	appVerCacheMu.Unlock()
}

// fetchAvailableVersion retrieves the latest available version of an app.
func fetchAvailableVersion(ctx context.Context, name string, meta appCatalogItem) string {
	// Binary / API-versioned installs (Go, Rust)
	if meta.LatestAPI != "" {
		rc, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(rc, "GET", meta.LatestAPI, nil)
		if err != nil {
			return ""
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return ""
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if meta.LatestRx != "" {
			rx := regexp.MustCompile(`(?ms)` + meta.LatestRx)
			if m := rx.FindStringSubmatch(string(body)); len(m) >= 2 {
				return strings.TrimSpace(m[1])
			}
		}
		return strings.TrimSpace(strings.Split(string(body), "\n")[0])
	}

	// Package manager query
	checkPkg := appCheckPkgForDistro(meta)
	if checkPkg == "" {
		return ""
	}
	return pkgAvailableVersion(checkPkg)
}

// pkgAvailableVersion queries the system package manager for the candidate version.
func pkgAvailableVersion(pkg string) string {
	switch pkgFamily {
	case "apt":
		out, err := exec.Command("apt-cache", "policy", pkg).Output()
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "Candidate:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 && parts[1] != "(none)" {
					return parts[1]
				}
			}
		}
	case "dnf", "yum":
		out, err := exec.Command(pkgFamily, "info", "--quiet", pkg).Output()
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(out), "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "Version") {
				parts := strings.SplitN(t, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	case "pacman":
		out, err := exec.Command("pacman", "-Si", pkg).Output()
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(out), "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "Version") {
				parts := strings.SplitN(t, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	case "zypper":
		out, err := exec.Command("zypper", "info", pkg).Output()
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "Version:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	return ""
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func appPkgsForDistro(meta appCatalogItem) []string {
	switch pkgFamily {
	case "apt":
		return meta.AptPkgs
	case "dnf", "yum":
		return meta.DnfPkgs
	case "pacman":
		return meta.PacmanPkgs
	case "zypper":
		return meta.ZypperPkgs
	}
	return nil
}

func appCheckPkgForDistro(meta appCatalogItem) string {
	switch pkgFamily {
	case "apt":
		return meta.AptCheckPkg
	case "dnf", "yum":
		return meta.DnfCheckPkg
	case "pacman":
		if len(meta.PacmanPkgs) > 0 {
			return meta.PacmanPkgs[0]
		}
	case "zypper":
		if len(meta.ZypperPkgs) > 0 {
			return meta.ZypperPkgs[0]
		}
	}
	return ""
}

// pkgInstallArgs builds the install command args for non-apt package managers.
func pkgInstallArgs(pkgs []string) ([]string, string) {
	switch pkgFamily {
	case "dnf", "yum":
		return append([]string{"-y", "install"}, pkgs...), pkgFamily
	case "pacman":
		return append([]string{"-Sy", "--noconfirm"}, pkgs...), "pacman"
	case "zypper":
		return append([]string{"-n", "install"}, pkgs...), "zypper"
	}
	return append([]string{"-y", "install"}, pkgs...), "apt-get"
}

// findAdminUser returns the most likely admin user for per-user installs (Rust, Zsh, etc.)
func findAdminUser() string {
	// Prefer SUDO_USER from environment — set when running as root via sudo
	if u := os.Getenv("SUDO_USER"); u != "" && u != "root" {
		return u
	}
	// Scan /home for the first real user directory
	entries, err := os.ReadDir("/home")
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			return e.Name()
		}
	}
	return ""
}

// rustCandidateDirs returns home directory paths to scan for Rust/cargo installs.
func rustCandidateDirs() []string {
	dirs := []string{"/root"}
	entries, _ := os.ReadDir("/home")
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, "/home/"+e.Name())
		}
	}
	return dirs
}

// versionEqual compares two version strings, stripping common package manager
// epoch prefixes (e.g. "2:1.2.3" vs "1.2.3") and Debian revision suffixes
// (e.g. "1.2.3-1ubuntu7.3" vs "1.2.3") before comparing.
func versionEqual(a, b string) bool {
	strip := func(v string) string {
		// Strip epoch prefix (e.g., "1:" from "1:2.43.0-1ubuntu7.3")
		if i := strings.Index(v, ":"); i >= 0 {
			v = v[i+1:]
		}
		// Strip Debian revision suffix (e.g., "-1ubuntu7.3" from "2.43.0-1ubuntu7.3")
		// But be careful: some version strings like "v1.2.3" or "go1.20" may have hyphens
		// We only strip the Debian revision if it looks like a package version
		// Debian revision is typically after the last hyphen followed by a number
		if i := strings.LastIndex(v, "-"); i > 0 {
			// Check if what follows the hyphen starts with a digit (Debian revision pattern)
			suffix := v[i+1:]
			if len(suffix) > 0 && suffix[0] >= '0' && suffix[0] <= '9' {
				v = v[:i]
			}
		}
		return v
	}
	return strip(a) == strip(b)
}

// appLog appends a single line to the live operation log.
func appLog(line string) {
	appOpMu.Lock()
	appOpLogs = append(appOpLogs, line)
	appOpMu.Unlock()
}

// appLogMulti splits multi-line command output and appends each non-empty line.
func appLogMulti(out string) {
	appOpMu.Lock()
	for _, line := range strings.Split(out, "\n") {
		if line != "" {
			appOpLogs = append(appOpLogs, line)
		}
	}
	appOpMu.Unlock()
}
