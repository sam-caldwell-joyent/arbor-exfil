package cmd

import (
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/require"
)

func TestLoadManifest_Success_AndAliasCmd_Dedicated(t *testing.T) {
    tmp := t.TempDir()
    p := writeTemp(t, tmp, "m.yaml", `
name: A
description: B
commands:
  - cmd: show version
    args: ["arg1", "arg 2"]
`)
    mf, err := loadManifest(p)
    require.NoError(t, err)
    require.Equal(t, "A", mf.Name)
    require.Equal(t, 1, len(mf.Commands))
    require.Equal(t, "show version", mf.Commands[0].Command)
    require.Equal(t, "show version arg1 'arg 2'", mf.Commands[0].line())
}

func TestLoadManifest_ValidationErrors_Dedicated(t *testing.T) {
    tmp := t.TempDir()
    // Missing name
    p1 := writeTemp(t, tmp, "m1.yaml", `
description: D
commands:
  - command: x
`)
    _, err := loadManifest(p1)
    require.Error(t, err)

    // Missing description
    p2 := writeTemp(t, tmp, "m2.yaml", `
name: N
commands:
  - command: x
`)
    _, err = loadManifest(p2)
    require.Error(t, err)

    // Missing command
    p3 := writeTemp(t, tmp, "m3.yaml", `
name: N
description: D
commands:
  - command: ""
`)
    _, err = loadManifest(p3)
    require.Error(t, err)
}

func TestLoadManifest_FileNotFound_Dedicated(t *testing.T) {
    _, err := loadManifest(filepath.Join(t.TempDir(), "missing.yaml"))
    require.Error(t, err)
}

