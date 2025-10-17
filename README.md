# arbor-exfil

CLI to connect to an Arbor TMS leader (ArbOS) over SSH, run a list of commands from a YAML manifest, and capture 
responses to a structured YAML report file.

## Usage

### Build:
```
# YAML example
name: Example Arbor Exfil
description: Run read-only ArbOS commands to collect diagnostics
generated: 2025-10-09T18:00:00Z
discovery:
  hosts_content: |
    127.0.0.1 localhost
    10.0.0.5 arbos-leader
  discovered_hosts:
    - 127.0.0.1
    - 10.0.0.5
runs:
  - host: 10.0.0.5
    results:
      - title: Show version
        command: show version
        shell: /bin/comsh
        exit_code: 0
        output: |
          ...device output...
```
make build
```

### Run:
```
./arbor-exfil --target tms.example.com:22 \
              --user arbor \
              --manifest manifests/inspection_report.yaml \
              --out output.txt \
              --known-hosts ~/.ssh/known_hosts`
```

If your manifest provides `ssh_host`, you can omit `--target` and `--user` and the tool will use those defaults
from the manifest (CLI flags still take precedence if provided):

```
./arbor-exfil --manifest manifests/inspection_report.yaml \
              --out output.txt \
              --strict-host-key=false
```

#### Key flags
  - `--target`: FQDN/IP:port of ArbOS (e.g., `10.0.0.5:22`).
  - `--manifest`: Path to YAML manifest with commands.
  - `--out`: Output file path; created if missing.
  - `--user`: SSH username.
  - `--password`: SSH password; can also set `ARBOR_EXFIL_PASSWORD`.
  - `--key`: Path to SSH private key; optional alternative to password.
  - `--passphrase`: Passphrase for encrypted private key; or `ARBOR_EXFIL_PASSPHRASE`.
  - `--known-hosts`: Path to `known_hosts` for host key verification.
  - `--strict-host-key`: Enforce host key verification (default true). Set to `false` to accept any host key.
  - `--cmd-timeout`: Per-command timeout, e.g., `30s` (0 = no timeout).
  - `--conn-timeout`: SSH connection timeout (default 15s).

#### Manifest format
The manifest is a YAML file with metadata, optional SSH defaults, and a list of commands.

Fields:
- `name` (string, required): Report name.
- `description` (string, required): Report description.
- `ssh_host` (object, optional): Default SSH connection when CLI flags are not provided.
  - `ip` (string): IP or host (may include `:port`; if omitted, `:22` is assumed).
  - `user` (string): SSH username.
- `commands` (array, required but may be empty): Steps to run. If empty or omitted, the tool performs discovery-only (see below).
  - `title` (string, optional): Section heading written before command output.
  - `command` (string, required when present): Base command to execute.
  - `args` (array[string], optional): Arguments appended to the command (safely quoted).
  - `timeout` (string duration, optional): Per-command timeout like `30s`; overrides `--cmd-timeout`.
  - `shell` (string, required for each command): Per-command shell path used with the sudo wrapper.

Example:
```
name: Example Arbor Exfil
description: Run read-only ArbOS commands to collect diagnostics
ssh_host:
  ip: 10.0.0.5        # or 10.0.0.5:2222 to use a non-default port
  user: arbor-exfil
commands:
  - title: Show version
    command: show version
    shell: /bin/comsh
    args: []
  - title: Show device status
    command: show device status
    shell: /bin/comsh
  - title: Show routes
    command: show routes
    shell: /bin/comsh
    timeout: 45s
```

### Output Format (YAML)
The output file is structured YAML with discovery and per-host results:

```
Name: Example Arbor Exfil
Description: Run read-only ArbOS commands to collect diagnostics
Generated: 2025-10-09T18:00:00Z
Command Count: 3
================================================================================
--------------------------------------------------------------------------------
Command: show version
Exit Code: 0
Output:
---8<---
...device output...
---8<---
```

### Host Discovery
- Before executing any commands, the tool connects to the leader (`ssh_host.ip`) and captures `/etc/hosts`.
- If there are commands to run, the full `/etc/hosts` contents are written to the report under a section titled `Host Discovery (/etc/hosts)`.
- If `commands` is empty or omitted, the report will instead include only a `Discovered Hosts` section listing deduplicated IPs parsed from `/etc/hosts`.
- The first column is parsed as IP addresses to build a deduplicated list of “child hosts”.
- Loopback addresses: IPv4 loopback (`127.0.0.1`) is included so the leader itself is targeted as a child host. IPv6 loopback (`::1`) is filtered out.
- For every child host, each command is executed using:
  - `sudo -u admin --shell $shell -c '$command'`
  - `$shell` is `commands[].shell` from the manifest (required; no fallback)
  - `$command` is the fully assembled `command` + `args`

#### Discovery-only mode
To perform host discovery without running any commands, provide an empty commands list (or omit it):

```
name: Discovery Only
description: Capture /etc/hosts and list hosts
ssh_host:
  ip: 10.0.0.5
  user: arbor-exfil
commands: []
```

The output will include YAML like:

```
... header ...
--------------------------------------------------------------------------------
Discovered Hosts:
127.0.0.1
10.0.0.5
10.0.0.23
```

### No-op mode
- Use `--noop` to preview what would run without executing on the remote.
- Writes the full wrapped command lines to `debug.out` and exits.
- Still performs host discovery to determine the set of child hosts.
- If `commands` is empty, discovery-only runs and no `debug.out` is produced (there are no planned commands).

### Notes
- Authentication supports: password, private key, or SSH agent if available.
- By default, strict host key checking is enabled and reads `--known-hosts`.
- If you don’t have `known_hosts` or are testing in a lab, set `--strict-host-key=false`.
- Commands execute in a single persistent SSH session so state (e.g., working directory, env) persists across commands.
- A PTY is requested for the session so commands that expect a TTY (colorized output, full-screen tools) behave the same as a real terminal.
- If a command times out, the SSH connection is closed, reconnected once, and a new persistent session is established
  before continuing with remaining commands.
- When `ssh_host` is present in the manifest, it is used only when the corresponding CLI flags are omitted; CLI values take precedence. If `ssh_host.ip` lacks a port, `:22` is assumed.
