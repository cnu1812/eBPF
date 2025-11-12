#!/bin/bash

# --- Configuration ---
SCRIPT_DIR=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
FLAME_DIR="${SCRIPT_DIR}/FlameGraph"
WWW_DIR="${SCRIPT_DIR}/www/html"
# --- End Configuration ---

echo "Starting real-time profiler..."
echo "Looking for 'slow_app.py' process..."
TARGET_PID=$(pgrep -f slow_app.py)
if [ -z "$TARGET_PID" ]; then
    echo "ERROR: 'slow_app.py' process not found."
    exit 1
fi

echo "Successfully found target PID: $TARGET_PID"
echo "Starting eBPF loop... Press Ctrl+C to stop."

# --- The Main Loop ---
while true; do
    # 1. Capture stacks for 5 seconds
    #    NEW: We are using the PID filter (which worked)
    #    NEW: We are using a 5-second interval
    sudo bpftrace -B none -e "profile:hz:99 /pid == $TARGET_PID/ { @stacks[ustack(12)] = count(); } interval:s:5 { print(@stacks); exit(); }" > $WWW_DIR/stacks.out 2> $WWW_DIR/stacks.err

    # Check for errors
    if [ -s "$WWW_DIR/stacks.err" ]; then
        echo "--- BPFTRACE ERROR ---"
        cat $WWW_DIR/stacks.err
        echo "----------------------"
        > $WWW_DIR/stacks.err
    fi

    # 2. Fold the stacks
    $FLAME_DIR/stackcollapse-bpftrace.pl $WWW_DIR/stacks.out > $WWW_DIR/stacks.folded

    # 3. Check if we captured anything
    if [ ! -s "$WWW_DIR/stacks.folded" ]; then
        echo "No stacks captured. (Waiting for load...)"
        # Create a placeholder "Waiting" SVG
        echo '<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="200" viewBox="0 0 1200 200" style="background-color: #333;">
            <text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" font-family="sans-serif" font-size="24" fill="#aaa">
                Waiting for CPU activity from slow_app.py (PID: '$TARGET_PID')...
            </text>
        </svg>' > $WWW_DIR/profile.svg
    else
        # 4. Render the SVG
        $FLAME_DIR/flamegraph.pl $WWW_DIR/stacks.folded > $WWW_DIR/profile.tmp
        mv $WWW_DIR/profile.tmp $WWW_DIR/profile.svg
        echo "Updated profile.svg at $(date +%T)"
    fi
done
