#!/usr/bin/env bash
# Opens a tmux session with live logs from all Stageflow containers
# Usage: ./scripts/logs.sh [session-name]

set -euo pipefail

SESSION_NAME="${1:-stageflow-logs}"
CONTAINER_PATTERN="${STAGEFLOW_CONTAINER_PATTERN:-^(systemd-)?stageflow-}"

# Get running stageflow containers
get_containers() {
  podman ps --format '{{.Names}}' | rg -e "${CONTAINER_PATTERN}" | sort
}

# Kill existing session if it exists
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
  echo "Killing existing session: $SESSION_NAME"
  tmux kill-session -t "$SESSION_NAME"
fi

# Get container list
mapfile -t CONTAINERS < <(get_containers)

if [[ ${#CONTAINERS[@]} -eq 0 ]]; then
  echo "No containers found matching pattern: ${CONTAINER_PATTERN}"
  exit 1
fi

echo "Found ${#CONTAINERS[@]} containers: ${CONTAINERS[*]}"

# Create new session with first container
FIRST="${CONTAINERS[0]}"
SHORT_FIRST="$FIRST"
SHORT_FIRST="${SHORT_FIRST#systemd-}"
SHORT_FIRST="${SHORT_FIRST#stageflow-}"
tmux new-session -d -s "$SESSION_NAME" -n logs \
  "printf '\\033[1;36m=== %s ===\\033[0m\\n' '$SHORT_FIRST'; podman logs -f '$FIRST' 2>&1; read"

# Add remaining containers in split panes
for ((i=1; i<${#CONTAINERS[@]}; i++)); do
  CONTAINER="${CONTAINERS[$i]}"
  SHORT_NAME="$CONTAINER"
  SHORT_NAME="${SHORT_NAME#systemd-}"
  SHORT_NAME="${SHORT_NAME#stageflow-}"
  
  # Split the current window
  tmux split-window -t "$SESSION_NAME:logs" \
    "printf '\\033[1;36m=== %s ===\\033[0m\\n' '$SHORT_NAME'; podman logs -f '$CONTAINER' 2>&1; read"
  
  # Rebalance after each split
  tmux select-layout -t "$SESSION_NAME:logs" tiled
done

# Enable pane titles
tmux set-option -t "$SESSION_NAME" pane-border-status top
tmux set-option -t "$SESSION_NAME" pane-border-format " #{pane_index}: #{pane_title} "

# Set pane titles
for ((i=0; i<${#CONTAINERS[@]}; i++)); do
  CONTAINER="${CONTAINERS[$i]}"
  SHORT_NAME="$CONTAINER"
  SHORT_NAME="${SHORT_NAME#systemd-}"
  SHORT_NAME="${SHORT_NAME#stageflow-}"
  tmux select-pane -t "$SESSION_NAME:logs.$i" -T "$SHORT_NAME" 2>/dev/null || true
done

echo "Attaching to tmux session: $SESSION_NAME"
echo "Tips: Ctrl-b + arrow keys to navigate, Ctrl-b + z to zoom pane"

# Attach to session
exec tmux attach-session -t "$SESSION_NAME"
