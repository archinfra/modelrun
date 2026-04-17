package collect

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"modelrun/backend/internal/domain"
)

const defaultNetdataEndpoint = "http://127.0.0.1:19999"

type NetdataStatus struct {
	Endpoint  string `json:"endpoint"`
	Reachable bool   `json:"reachable"`
	Message   string `json:"message"`
	Hostname  string `json:"hostname,omitempty"`
	Version   string `json:"version,omitempty"`
}

type netdataInfo struct {
	Hostname string `json:"hostname"`
	HostName string `json:"host_name"`
	Version  string `json:"version"`
}

func DefaultNetdataEndpoint() string {
	return defaultNetdataEndpoint
}

func (c *Collector) NetdataStatus(server domain.ServerConfig, jump *SSHConfig) (NetdataStatus, error) {
	endpoint := normalizeNetdataEndpoint(server.NetdataEndpoint)
	if IsMockServer(server) {
		return NetdataStatus{
			Endpoint:  endpoint,
			Reachable: true,
			Message:   "mock netdata dashboard is reachable",
			Hostname:  firstNonEmpty(server.Name, server.Host, "mock-node"),
			Version:   "1.45.3",
		}, nil
	}

	client, closeFn, err := c.DialSSH(server, jump)
	if err != nil {
		return NetdataStatus{}, err
	}
	defer closeFn()

	return netdataStatusFromClient(client, endpoint), nil
}

func netdataStatusFromClient(client sshRunner, endpoint string) NetdataStatus {
	status := NetdataStatus{Endpoint: endpoint}
	info, err := fetchNetdataInfo(client, endpoint)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	status.Reachable = true
	status.Message = "netdata dashboard is reachable"
	status.Hostname = firstNonEmpty(info.Hostname, info.HostName)
	status.Version = strings.TrimSpace(info.Version)
	return status
}

func fetchNetdataInfo(client sshRunner, endpoint string) (netdataInfo, error) {
	infoURL := strings.TrimRight(endpoint, "/") + "/api/v1/info"
	command := "if command -v curl >/dev/null 2>&1; then " +
		"curl -fsS --max-time 3 " + shellQuote(infoURL) + "; " +
		"elif command -v wget >/dev/null 2>&1; then " +
		"wget -q -T 3 -O - " + shellQuote(infoURL) + "; " +
		"else echo 'curl or wget is required to query netdata' >&2; exit 127; fi"

	out, err := run(client, command)
	if err != nil {
		return netdataInfo{}, err
	}
	var info netdataInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return netdataInfo{}, fmt.Errorf("parse netdata info: %w", err)
	}
	return info, nil
}

func normalizeNetdataEndpoint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultNetdataEndpoint
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return defaultNetdataEndpoint
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "http"
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func NetdataTarget(endpoint string) (string, string, error) {
	parsed, err := url.Parse(normalizeNetdataEndpoint(endpoint))
	if err != nil {
		return "", "", err
	}
	if parsed.Host == "" {
		return "", "", errors.New("netdata endpoint host is empty")
	}
	return parsed.Scheme, parsed.Host, nil
}
