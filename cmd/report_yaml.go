package cmd

import (
    "bufio"
    "bytes"
    "io"
    "time"
    "gopkg.in/yaml.v3"
)

// yamlReport is the top-level structure serialized to the output YAML file.
// It mirrors the high-level expectations of consumers: metadata, discovery
// information, and per-host command results.
type yamlReport struct {
    Name        string         `yaml:"name"`
    Description string         `yaml:"description"`
    Generated   string         `yaml:"generated"`
    Discovery   *yamlDiscovery `yaml:"discovery,omitempty"`
    Runs        []yamlRun      `yaml:"runs,omitempty"`
}

// yamlDiscovery captures discovery content and the derived list of IPs.
type yamlDiscovery struct {
    HostsContent   string   `yaml:"hosts_content,omitempty"`
    DiscoveredHosts []string `yaml:"discovered_hosts"`
}

// yamlRun groups the results for a single host. When discovery yields no
// hosts, a single run with empty host is used to represent the leader context.
type yamlRun struct {
    Host    string           `yaml:"host,omitempty"`
    Results []yamlCmdResult  `yaml:"results"`
}

// yamlCmdResult records the outcome of a single command execution.
type yamlCmdResult struct {
    Title    string `yaml:"title,omitempty"`
    Command  string `yaml:"command"`
    Shell    string `yaml:"shell,omitempty"`
    Timeout  string `yaml:"timeout,omitempty"`
    ExitCode int    `yaml:"exit_code"`
    Error    string `yaml:"error,omitempty"`
    Output   string `yaml:"output"`
}

// newYAMLReport constructs a report seeded with manifest metadata and a
// generated timestamp.
func newYAMLReport(mf *manifest) *yamlReport {
    return &yamlReport{
        Name:        mf.Name,
        Description: mf.Description,
        Generated:   time.Now().Format(time.RFC3339),
    }
}

// ensureDiscovery lazily initializes the discovery section.
func (r *yamlReport) ensureDiscovery() {
    if r.Discovery == nil {
        r.Discovery = &yamlDiscovery{DiscoveredHosts: []string{}}
    }
}

// setDiscovery fills in discovery information. When includeContent is true,
// the raw /etc/hosts content is embedded; otherwise only the host list is set.
func (r *yamlReport) setDiscovery(hostsContent []byte, hosts []string, includeContent bool) {
    r.ensureDiscovery()
    if includeContent {
        r.Discovery.HostsContent = string(hostsContent)
    }
    r.Discovery.DiscoveredHosts = hosts
}

// addRun finds or creates a run entry for the specified host.
func (r *yamlReport) addRun(host string) *yamlRun {
    // Find existing
    for i := range r.Runs {
        if r.Runs[i].Host == host {
            return &r.Runs[i]
        }
    }
    r.Runs = append(r.Runs, yamlRun{Host: host})
    return &r.Runs[len(r.Runs)-1]
}

// addResult appends a command result beneath the specified host run.
func (r *yamlReport) addResult(host string, res yamlCmdResult) {
    run := r.addRun(host)
    run.Results = append(run.Results, res)
}

// writeYAMLReport serializes the report to YAML with indentation and writes to
// the provided writer in a buffered manner for efficiency.
func writeYAMLReport(w io.Writer, r *yamlReport) error {
    var buf bytes.Buffer
    enc := yaml.NewEncoder(&buf)
    enc.SetIndent(2)
    if err := enc.Encode(r); err != nil {
        _ = enc.Close()
        return err
    }
    _ = enc.Close()
    bw := bufio.NewWriter(w)
    if _, err := bw.Write(buf.Bytes()); err != nil {
        return err
    }
    return bw.Flush()
}
