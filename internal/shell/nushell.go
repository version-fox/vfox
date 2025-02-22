package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/version-fox/vfox/internal/env"
)

type nushell struct{}

var Nushell = nushell{}

const nushellConfig = `
# vfox configuration
# this make sure this configuration is up to date when you open a new shell
^'{{.SelfPath}}' activate nushell $nu.default-config-dir | ignore

export-env {
  def --env updateVfoxEnvironment [] {
    let envData = (^'{{.SelfPath}}' env -s nushell --full | from json)
	if ($envData | is-empty) {
      return
    }
    load-env $envData.envsToSet
    hide-env ...$envData.envsToUnset
  }
  $env.config = ($env.config | upsert hooks.pre_prompt {
    let currentValue = ($env.config | get -i hooks.pre_prompt)
    if $currentValue == null {
      [{updateVfoxEnvironment}]
    } else {
      $currentValue | append {updateVfoxEnvironment}
    }
  })
  $env.__VFOX_SHELL = 'nushell'
  $env.__VFOX_PID = $nu.pid
  ^'{{.SelfPath}}' env --cleanup | ignore
  updateVfoxEnvironment
}
`

// We create a `vfox.nu“ in the `$nu.default-config-dir“
// Activate implements shell.Activate will generate a script to the `vfox.nu` file.
// This script does the following:
//
// 1. Sets up a [pre_prompt hook] to update the environment variables when needed.
// 2. Initializes the __VFOX_SHELL and __VFOX_PID environment variables.
// 3. Runs the vfox cleanup task.
// 4. Updates the environment variables.
//
// [pre_prompt hook]: https://www.nushell.sh/book/hooks.html
func (n nushell) Activate(config ActivateConfig) (string, error) {
	if len(config.Args) == 0 {
		return "", fmt.Errorf("config path is required")
	}

	// write file to config
	targetPath := filepath.Join(config.Args[0], "vfox.nu")

	nushellConfig := strings.ReplaceAll(nushellConfig, "\n", env.Newline)
	nushellConfig = strings.ReplaceAll(nushellConfig, "{{.SelfPath}}", config.SelfPath)

	if err := os.WriteFile(targetPath, []byte(nushellConfig), 0755); err != nil {
		return "", fmt.Errorf("failed to write file: %s", err)
	}

	return `source ($nu.default-config-dir | path join "vfox.nu")` + env.Newline, nil
}

// nushellExportData is used to create a JSON representation of the environment variables to be set and unset.
type nushellExportData struct {
	EnvsToSet   map[string]any `json:"envsToSet"`
	EnvsToUnset []string       `json:"envsToUnset"`
}

// Export implements shell.Export by creating a JSON representation of the environment variables to be set and unset.
// Nushell can then convert this JSON string to a [record] using the [from json] command, so it can load and unload the
// environment variables using the [load-env] and [hide-env] commands.
//
// This approach is required for Nushell because it does not support eval-like functionality. For more background
// information on this, see the article [How Nushell Code Gets Run].
//
// [record]: https://www.nushell.sh/lang-guide/chapters/types/basic_types/record.html
// [from json]: https://www.nushell.sh/commands/docs/from_json.html
// [load-env]: https://www.nushell.sh/commands/docs/load-env.html
// [hide-env]: https://www.nushell.sh/commands/docs/hide-env.html
// [How Nushell Code Gets Run]: https://www.nushell.sh/book/how_nushell_code_gets_run.html
func (n nushell) Export(envs env.Vars) string {
	exportData := nushellExportData{
		EnvsToSet:   make(map[string]any),
		EnvsToUnset: make([]string, 0),
	}

	for key, value := range envs {
		if key == "PATH" { // Convert from string to list.
			if value == nil {
				value = new(string)
			}
			pathEntries := filepath.SplitList(*value)
			exportData.EnvsToSet[key] = pathEntries
		} else {
			if value == nil {
				exportData.EnvsToUnset = append(exportData.EnvsToUnset, key)
			} else {
				exportData.EnvsToSet[key] = *value
			}
		}
	}

	exportJson, err := json.Marshal(exportData)
	if err != nil {
		fmt.Printf("Failed to marshal export data: %s\n", err)
		return ""
	}

	return string(exportJson)
}
