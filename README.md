# arbor-exfil

CLI to connect to an Arbor TMS leader (ArbOS) over SSH, run a list of commands from a YAML manifest, and capture 
responses to a structured YAML report file.

## Usage

### Build
```
make build
```
or build a local Docker image:
```
make docker
```

### Run:
```
./arbor-exfil run --target tms.example.com:22 \
              --user arbor \
              --manifest manifests/inspection_report.yaml \
              --out output.yaml \
              --known-hosts ~/.ssh/known_hosts
```

If your manifest provides `ssh_host`, you can omit `--target` and `--user` and the tool will use those defaults
from the manifest (CLI flags still take precedence if provided):

```
./arbor-exfil run --manifest manifests/inspection_report.yaml \
              --out output.yaml \
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
  - `--install-pubkey`: Path to SSH public key for `install` (install only).

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
The output file is structured YAML with discovery and per-host results, e.g.:

```
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
name: Discovery Only
description: Capture /etc/hosts and list hosts
generated: 2025-10-09T18:00:00Z
discovery:
  discovered_hosts:
    - 127.0.0.1
    - 10.0.0.5
    - 10.0.0.23
```

### Verify manifest
Validate a manifest without running any commands:

```
./arbor-exfil verify --manifest manifests/inspection_report.yaml
# prints: Manifest OK (or an error describing problems)
```

Validation includes required fields (`name`, `description`), that each listed command has a non-empty `command`, and when `commands` are present each has a non-empty `shell`.

### Install keys
Provision an `arbor-exfil` user and install a provided SSH public key on all discovered non-loopback hosts. The manifest’s `ssh_host` leader is used for host discovery; `--target` and `--user` may be omitted if present in the manifest.

```
./arbor-exfil install \
  --manifest manifests/inspection_report.yaml \
  --install-pubkey ~/.ssh/arbor-exfil.pub \
  --strict-host-key=false
```

Details:
- Connects to the leader to read `/etc/hosts` and parse IPs.
- Filters loopback addresses (both `127.*` and `::1`) for install only.
- Dials each discovered host using the provided SSH credentials, then:
  - Creates user `arbor-exfil` if missing (`useradd` or `adduser`).
  - Creates `~arbor-exfil/.ssh`, appends the given public key to `authorized_keys`, and fixes permissions.
- Uses `sudo -n` for non-interactive privilege elevation on the remote.

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

### yq examples
- List discovered hosts:
  - `yq '.discovery.discovered_hosts[]' output.yaml`
- List commands with host and exit code:
  - `yq '.runs[] as $r | $r.results[] | {host: $r.host, command: .command, exit: .exit_code}' output.yaml`
- Extract output for a specific titled command:
  - `yq -r '.runs[].results[] | select(.title=="Show version") | .output' output.yaml`
