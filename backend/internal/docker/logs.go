package docker

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var streamClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/var/run/docker.sock")
		},
	},
	Timeout: 0,
}

// ContainerLogsOptions controls which logs are fetched from Docker.
type ContainerLogsOptions struct {
	Stdout     bool
	Stderr     bool
	Since      string
	Until      string
	Tail       int
	Timestamps bool
	Follow     bool
}

// DockerLogLine is a parsed Docker log line with stream metadata.
type DockerLogLine struct {
	Stream    string `json:"stream"`
	Timestamp string `json:"timestamp,omitempty"`
	Text      string `json:"text"`
}

func buildLogsURL(id string, opts ContainerLogsOptions) (string, error) {
	values := url.Values{}
	if opts.Stdout {
		values.Set("stdout", "1")
	}
	if opts.Stderr {
		values.Set("stderr", "1")
	}
	if opts.Timestamps {
		values.Set("timestamps", "1")
	}
	if opts.Follow {
		values.Set("follow", "1")
	}
	if opts.Since != "" {
		values.Set("since", opts.Since)
	}
	if opts.Until != "" {
		values.Set("until", opts.Until)
	}
	if opts.Tail > 0 {
		values.Set("tail", fmt.Sprintf("%d", opts.Tail))
	}
	return "/containers/" + url.PathEscape(id) + "/logs?" + values.Encode(), nil
}

func ContainerLogs(id string, opts ContainerLogsOptions) ([]DockerLogLine, error) {
	if !opts.Stdout && !opts.Stderr {
		opts.Stdout = true
		opts.Stderr = true
	}
	url, err := buildLogsURL(id, opts)
	if err != nil {
		return nil, err
	}
	resp, err := socketClient.Get("http://docker" + url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return parseDockerLogStream(resp.Body, opts)
}

func ContainerLogsStream(ctx context.Context, id string, opts ContainerLogsOptions) (io.ReadCloser, error) {
	if !opts.Stdout && !opts.Stderr {
		opts.Stdout = true
		opts.Stderr = true
	}
	reqURL, err := buildLogsURL(id, opts)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker"+reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("docker: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return resp.Body, nil
}

func parseDockerLogStream(reader io.Reader, opts ContainerLogsOptions) ([]DockerLogLine, error) {
	buf := bufio.NewReader(reader)
	if !opts.Stdout || !opts.Stderr {
		return parseRawDockerStream(buf, opts)
	}
	peek, err := buf.Peek(8)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(peek) < 8 || (peek[0] != 0 && peek[0] != 1 && peek[0] != 2) || peek[1] != 0 || peek[2] != 0 || peek[3] != 0 {
		return parseRawDockerStream(buf, opts)
	}
	return parseMultiplexedDockerStream(buf)
}

func parseRawDockerStream(reader *bufio.Reader, opts ContainerLogsOptions) ([]DockerLogLine, error) {
	result := []DockerLogLine{}
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			trimmed := strings.TrimSuffix(line, "\n")
			if trimmed != "" {
				result = append(result, DockerLogLine{Stream: "stdout", Timestamp: extractTimestamp(trimmed), Text: trimmed})
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return result, nil
			}
			return nil, err
		}
	}
}

func parseMultiplexedDockerStream(reader *bufio.Reader) ([]DockerLogLine, error) {
	result := []DockerLogLine{}
	for {
		head := make([]byte, 8)
		if _, err := io.ReadFull(reader, head); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return result, nil
			}
			return nil, err
		}
		streamType := head[0]
		size := int(binary.BigEndian.Uint32(head[4:8]))
		if size <= 0 {
			continue
		}
		payload := make([]byte, size)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		stream := "stdout"
		if streamType == 2 {
			stream = "stderr"
		}
		text := strings.TrimSuffix(string(payload), "\n")
		for _, line := range strings.Split(text, "\n") {
			if line == "" {
				continue
			}
			result = append(result, DockerLogLine{Stream: stream, Timestamp: extractTimestamp(line), Text: line})
		}
	}
}

func extractTimestamp(line string) string {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	if _, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
		return parts[0]
	}
	return ""
}

func StreamDockerLogLines(ctx context.Context, reader io.ReadCloser, out chan<- DockerLogLine) error {
	defer reader.Close()
	buf := bufio.NewReader(reader)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		peek, err := buf.Peek(8)
		if err != nil && err != io.EOF {
			return err
		}
		if len(peek) >= 8 && (peek[0] == 0 || peek[0] == 1 || peek[0] == 2) && peek[1] == 0 && peek[2] == 0 && peek[3] == 0 {
			if err := streamNextMultiplexedChunk(buf, out); err != nil {
				return err
			}
			continue
		}
		line, err := buf.ReadString('\n')
		if line != "" {
			line = strings.TrimSuffix(line, "\n")
			out <- DockerLogLine{Stream: "stdout", Timestamp: extractTimestamp(line), Text: line}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func streamNextMultiplexedChunk(reader *bufio.Reader, out chan<- DockerLogLine) error {
	head := make([]byte, 8)
	if _, err := io.ReadFull(reader, head); err != nil {
		return err
	}
	streamType := head[0]
	size := int(binary.BigEndian.Uint32(head[4:8]))
	if size <= 0 {
		return nil
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return err
	}
	stream := "stdout"
	if streamType == 2 {
		stream = "stderr"
	}
	text := strings.TrimSuffix(string(payload), "\n")
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		out <- DockerLogLine{Stream: stream, Timestamp: extractTimestamp(line), Text: line}
	}
	return nil
}
