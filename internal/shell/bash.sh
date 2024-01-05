#!/usr/bin/bash

export VFOX_SHELL="bash"
export __MISE_ORIG_PATH="$PATH"

vfox() {
  if [ "$1" = "use" ]; then
    eval "$("{{.SelfPath}}" env -s bash)";
  else
    command "{{.SelfPath}}" "$@";
  fi
}

_vfox_hook() {
  local previous_exit_status=$?;
  trap -- '' SIGINT;
  eval "$("{{.SelfPath}}" env -s bash)";
  trap - SIGINT;
  return $previous_exit_status;
}

if [[ ";${{PROMPT_COMMAND:-}};" != *";_vfox_hook;"* ]]; then
  PROMPT_COMMAND="_vfox_hook${{PROMPT_COMMAND:+;$PROMPT_COMMAND}}"
fi