package cmd

import (
    "bufio"
    "bytes"
    "io"
    "time"
    "gopkg.in/yaml.v3"
)

type yamlReport struct {
    Name        string         `yaml:"name"`
    Description string         `yaml:"description"`
    Generated   string         `yaml:"generated"`
    Discovery   *yamlDiscovery `yaml:"discovery,omitempty"`
    Runs        []yamlRun      `yaml:"runs,omitempty"`
}

type yamlDiscovery struct {
    HostsContent   string   `yaml:"hosts_content,omitempty"`
    DiscoveredHosts []string `yaml:"discovered_hosts"`
}

type yamlRun struct {
    Host    string           `yaml:"host,omitempty"`
    Results []yamlCmdResult  `yaml:"results"`
}

type yamlCmdResult struct {
    Title    string `yaml:"title,omitempty"`
    Command  string `yaml:"command"`
    Shell    string `yaml:"shell,omitempty"`
    Timeout  string `yaml:"timeout,omitempty"`
    ExitCode int    `yaml:"exit_code"`
    Error    string `yaml:"error,omitempty"`
    Output   string `yaml:"output"`
}

func newYAMLReport(mf *manifest) *yamlReport {
    return &yamlReport{
        Name:        mf.Name,
        Description: mf.Description,
        Generated:   time.Now().Format(time.RFC3339),
    }
}

func (r *yamlReport) ensureDiscovery() {
    if r.Discovery == nil {
        r.Discovery = &yamlDiscovery{DiscoveredHosts: []string{}}
    }
}

func (r *yamlReport) setDiscovery(hostsContent []byte, hosts []string, includeContent bool) {
    r.ensureDiscovery()
    if includeContent {
        r.Discovery.HostsContent = string(hostsContent)
    }
    r.Discovery.DiscoveredHosts = hosts
}

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

func (r *yamlReport) addResult(host string, res yamlCmdResult) {
    run := r.addRun(host)
    run.Results = append(run.Results, res)
}

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

