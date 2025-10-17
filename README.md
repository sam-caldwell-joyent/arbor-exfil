# arbor-exfil

CLI to connect to an Arbor TMS leader (ArbOS) over SSH, run a list of commands from a YAML manifest, and capture 
responses to a text file.

## Usage

### Build:
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
- `commands` (array, required): Steps to run.
  - `title` (string, optional): Section heading written before command output.
  - `command` (string, required): Base command to execute.
  - `args` (array[string], optional): Arguments appended to the command (safely quoted).
  - `timeout` (string duration, optional): Per-command timeout like `30s`; overrides `--cmd-timeout`.
  - `shell` (string, required): Per-command shell path used with the sudo wrapper; required for execution.

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
    args: []
  - title: Show device status
    command: show device status
    shell: /bin/comsh   # optional; accepted but not used yet
  - title: Show routes
    command: show routes
    timeout: 45s
```

### Output format
The output file contains a header with manifest metadata, then a section per command:

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
- The full `/etc/hosts` contents are written to the report under a section titled `Host Discovery (/etc/hosts)`.
- The first column is parsed as IP addresses to build a deduplicated list of “child hosts”.
- Loopback addresses (e.g., `127.0.0.1`, `::1`) are intentionally included so the leader itself is targeted as a child host.
- For every child host, each command is executed using:
  - `sudo -u admin --shell $shell -c '$command'`
  - `$shell` is `commands[].shell` from the manifest (required; no fallback)
  - `$command` is the fully assembled `command` + `args`

### No-op mode
- Use `--noop` to preview what would run without executing on the remote.
- Writes the full wrapped command lines to `debug.out` and exits.
- Still performs host discovery to determine the set of child hosts.

### Notes
- Authentication supports: password, private key, or SSH agent if available.
- By default, strict host key checking is enabled and reads `--known-hosts`.
- If you don’t have `known_hosts` or are testing in a lab, set `--strict-host-key=false`.
- Commands execute in a single persistent SSH session so state (e.g., working directory, env) persists across commands.
- A PTY is requested for the session so commands that expect a TTY (colorized output, full-screen tools) behave the same as a real terminal.
- If a command times out, the SSH connection is closed, reconnected once, and a new persistent session is established
  before continuing with remaining commands.
- When `ssh_host` is present in the manifest, it is used only when the corresponding CLI flags are omitted; CLI values take precedence. If `ssh_host.ip` lacks a port, `:22` is assumed.
