package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var socketClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/var/run/docker.sock")
		},
	},
	Timeout: 5 * time.Second,
}

func get(path string, out any) error {
	resp, err := socketClient.Get("http://docker" + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// ContainerSummary is a trimmed representation of a Docker container.
type ContainerSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	State   string `json:"state"`
	Created int64  `json:"created"`
	Ports   string `json:"ports"`
}

type rawContainer struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	Status  string   `json:"Status"`
	State   string   `json:"State"`
	Created int64    `json:"Created"`
	Ports   []struct {
		IP          string `json:"IP"`
		PrivatePort uint16 `json:"PrivatePort"`
		PublicPort  uint16 `json:"PublicPort"`
		Type        string `json:"Type"`
	} `json:"Ports"`
}

// ListContainers returns all containers (running + stopped).
func ListContainers() ([]ContainerSummary, error) {
	var raw []rawContainer
	if err := get("/containers/json?all=true", &raw); err != nil {
		return nil, err
	}

	out := make([]ContainerSummary, 0, len(raw))
	for _, c := range raw {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		ports := ""
		for i, p := range c.Ports {
			if i > 0 {
				ports += ", "
			}
			if p.PublicPort > 0 {
				ports += fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type)
			} else {
				ports += fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
			}
		}

		out = append(out, ContainerSummary{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Created: c.Created,
			Ports:   ports,
		})
	}
	return out, nil
}

// ContainerStats holds realtime resource stats for a single container.
type ContainerStats struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	CPUPct float64 `json:"cpu_pct"`
	MemMB  float64 `json:"mem_mb"`
	MemPct float64 `json:"mem_pct"`
	NetRx  uint64  `json:"net_rx"`
	NetTx  uint64  `json:"net_tx"`
}

type rawStats struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     int    `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
}

// ContainerStatsOne fetches a one-shot stats snapshot for a single container.
func ContainerStatsOne(id string) (ContainerStats, error) {
	var raw rawStats
	if err := get("/containers/"+id+"/stats?stream=false", &raw); err != nil {
		return ContainerStats{}, err
	}

	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage)
	cpuPct := 0.0
	if sysDelta > 0 {
		ncpu := raw.CPUStats.OnlineCPUs
		if ncpu == 0 {
			ncpu = 1
		}
		cpuPct = (cpuDelta / sysDelta) * float64(ncpu) * 100.0
		cpuPct = float64(int(cpuPct*100+0.5)) / 100
	}

	memMB := float64(raw.MemoryStats.Usage) / (1024 * 1024)
	memPct := 0.0
	if raw.MemoryStats.Limit > 0 {
		memPct = float64(raw.MemoryStats.Usage) / float64(raw.MemoryStats.Limit) * 100
		memPct = float64(int(memPct*100+0.5)) / 100
	}

	var netRx, netTx uint64
	for _, n := range raw.Networks {
		netRx += n.RxBytes
		netTx += n.TxBytes
	}

	return ContainerStats{
		ID:     id,
		CPUPct: cpuPct,
		MemMB:  float64(int(memMB*100+0.5)) / 100,
		MemPct: memPct,
		NetRx:  netRx,
		NetTx:  netTx,
	}, nil
}

// DockerInfo holds high-level Docker daemon info.
type DockerInfo struct {
	Available         bool   `json:"available"`
	ServerVersion     string `json:"server_version"`
	ContainersTotal   int    `json:"containers_total"`
	ContainersRunning int    `json:"containers_running"`
	ImageCount        int    `json:"image_count"`
}

type rawInfo struct {
	ServerVersion     string `json:"ServerVersion"`
	Containers        int    `json:"Containers"`
	ContainersRunning int    `json:"ContainersRunning"`
	Images            int    `json:"Images"`
}

// post sends a POST to the Docker socket with no body and discards response.
func post(path string) error {
	resp, err := socketClient.Post("http://docker"+path, "application/json", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("docker: %s", resp.Status)
	}
	return nil
}

func postAndDecode(path string, out any) error {
	req, err := http.NewRequest(http.MethodPost, "http://docker"+path, nil)
	if err != nil {
		return err
	}
	resp, err := socketClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("docker: %s", resp.Status)
	}
	if out == nil {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

// StartContainer starts a stopped container by ID.
func StartContainer(id string) error {
	return post("/containers/" + id + "/start")
}

// StopContainer stops a running container by ID.
func StopContainer(id string) error {
	return post("/containers/" + id + "/stop")
}

// RestartContainer restarts a container by ID.
func RestartContainer(id string) error {
	return post("/containers/" + id + "/restart")
}

// PruneResult summarizes docker prune outcome.
type PruneResult struct {
	Kind           string `json:"kind"`
	Deleted        int    `json:"deleted"`
	ReclaimedSpace uint64 `json:"reclaimed_space"`
}

// PruneUnused runs one docker prune operation by kind.
// Supported kinds: containers, images, volumes, build, all.
func PruneUnused(kind string) (PruneResult, error) {
	if kind == "all" {
		total := PruneResult{Kind: "all"}
		for _, k := range []string{"containers", "images", "volumes", "build"} {
			res, err := PruneUnused(k)
			if err != nil {
				return PruneResult{}, err
			}
			total.Deleted += res.Deleted
			total.ReclaimedSpace += res.ReclaimedSpace
		}
		return total, nil
	}

	var path string
	switch kind {
	case "containers":
		path = "/containers/prune"
	case "images":
		path = "/images/prune"
	case "volumes":
		path = "/volumes/prune"
	case "build":
		path = "/build/prune?all=1"
	default:
		return PruneResult{}, fmt.Errorf("unsupported prune kind: %s", kind)
	}

	var raw map[string]any
	if err := postAndDecode(path, &raw); err != nil {
		return PruneResult{}, err
	}

	result := PruneResult{Kind: kind}
	if v, ok := raw["SpaceReclaimed"]; ok {
		switch n := v.(type) {
		case float64:
			if n > 0 {
				result.ReclaimedSpace = uint64(n)
			}
		}
	}

	deletedFields := []string{"ContainersDeleted", "VolumesDeleted", "ImagesDeleted", "CachesDeleted"}
	for _, f := range deletedFields {
		if v, ok := raw[f]; ok {
			if arr, ok := v.([]any); ok {
				result.Deleted += len(arr)
			}
		}
	}

	return result, nil
}

func Info() DockerInfo {
	var raw rawInfo
	if err := get("/info", &raw); err != nil {
		return DockerInfo{Available: false}
	}
	return DockerInfo{
		Available:         true,
		ServerVersion:     raw.ServerVersion,
		ContainersTotal:   raw.Containers,
		ContainersRunning: raw.ContainersRunning,
		ImageCount:        raw.Images,
	}
}
