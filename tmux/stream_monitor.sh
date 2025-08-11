#!/bin/bash

# Log streaming script for Dolly tmux session manager
# Usage: stream_monitor.sh <session_name> [--grep keyword1 keyword2 ... --] <pane_id1> [pane_id2] ...

if [ $# -lt 2 ]; then
    echo "Usage: $0 <session_name> [--grep keyword1 keyword2 ... --] <pane_id1> [pane_id2] ..."
    exit 1
fi

SESSION_NAME="$1"
shift

# Parse grep keywords if provided
GREP_KEYWORDS=()
if [ "$1" = "--grep" ]; then
    shift
    while [ "$1" != "--" ] && [ $# -gt 0 ]; do
        GREP_KEYWORDS+=("$1")
        shift
    done
    if [ "$1" = "--" ]; then
        shift
    fi
fi

PANES=("$@")

clear
echo "=== Log Streaming Started for Session: $SESSION_NAME ==="
echo "Streaming from ${#PANES[@]} panes"
if [ ${#GREP_KEYWORDS[@]} -gt 0 ]; then
    echo "Filtering for keywords: ${GREP_KEYWORDS[*]}"
fi
echo "=================================="
echo ""

# Simple approach: use temporary files to track last line count for each pane
TEMP_DIR="/tmp/dolly_stream_${SESSION_NAME}"
mkdir -p "$TEMP_DIR"

# Initialize last line tracking for each pane
for pane in "${PANES[@]}"; do
    echo "0" > "${TEMP_DIR}/${pane//[^a-zA-Z0-9]/_}"
done

echo "Starting continuous log monitoring..."
echo "Press Ctrl+C to stop"
echo ""

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "Log streaming stopped."
    rm -rf "$TEMP_DIR"
    exit 0
}

# Trap Ctrl+C
trap cleanup INT

while true; do
    for pane in "${PANES[@]}"; do
        # Get current content from pane
        content=$(tmux capture-pane -t "$pane" -p 2>/dev/null)
        
        if [ -n "$content" ]; then
            # Count lines
            current_lines=$(echo "$content" | wc -l | tr -d ' ')
            
            # Get last line count from file
            pane_file="${TEMP_DIR}/${pane//[^a-zA-Z0-9]/_}"
            last_line_count=$(cat "$pane_file" 2>/dev/null || echo "0")
            
            # If there are new lines, show them
            if [ "$current_lines" -gt "$last_line_count" ]; then
                # Calculate how many new lines to show
                new_line_count=$((current_lines - last_line_count))
                
                # Get only the new lines
                new_content=$(echo "$content" | tail -n "$new_line_count")
                
                # Apply grep filtering if keywords are specified
                if [ ${#GREP_KEYWORDS[@]} -gt 0 ]; then
                    filtered_content=""
                    for keyword in "${GREP_KEYWORDS[@]}"; do
                        # Use grep -i for case-insensitive matching
                        keyword_matches=$(echo "$new_content" | grep -i "$keyword" || true)
                        if [ -n "$keyword_matches" ]; then
                            filtered_content="$filtered_content$keyword_matches"$'\n'
                        fi
                    done
                    new_content="$filtered_content"
                fi
                
                # Only show if there's content to display
                if [ -n "$new_content" ] && [ "$new_content" != $'\n' ]; then
                    echo "[$pane] $(date '+%H:%M:%S'):"
                    echo "$new_content"
                    echo ""
                fi
                
                # Update the last line count regardless of filtering
                echo "$current_lines" > "$pane_file"
            fi
        fi
    done
    
    sleep 1
done