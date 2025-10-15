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
The manifest is a YAML file with metadata and a list of commands. Each command has a `command` and optional `args` 
array. A per-command `timeout` may override the global `--cmd-timeout`.

Example:
```
name: Example Arbor Exfil
description: Run read-only ArbOS commands to collect diagnostics
commands:
  - command: show version
    args: []
  - command: show device status
  - command: show routes
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

### Notes
- Authentication supports: password, private key, or SSH agent if available.
- By default, strict host key checking is enabled and reads `--known-hosts`.
- If you don’t have `known_hosts` or are testing in a lab, set `--strict-host-key=false`.
- Commands execute in a single persistent SSH session so state (e.g., working directory, env) persists across commands.
- A PTY is requested for the session so commands that expect a TTY (colorized output, full-screen tools) behave the same as a real terminal.
- If a command times out, the SSH connection is closed, reconnected once, and a new persistent session is established
  before continuing with remaining commands.
