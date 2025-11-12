#!/bin/bash
#
# IMPORTANT: You must run this script with sudo:
# sudo ./realtime-calls.sh
#

# --- Check for 'bc' dependency ---
if ! command -v bc &> /dev/null; then
    echo "ERROR: 'bc' is not installed."
    echo "Attempting to install 'bc'..."
    if apt-get update && apt-get install -y bc; then
        echo "'bc' installed successfully."
    else
        echo "Failed to install 'bc'. Please install it manually."
        exit 1
    fi
fi

# --- Configuration ---
echo "Starting real-time call counter..."
echo "Looking for 'slow_app.py' process..."

TARGET_PID=$(pgrep -f slow_app.py)
if [ -z "$TARGET_PID" ]; then
    echo "ERROR: 'slow_app.py' process not found. Is it running?"
    exit 1
fi

TARGET_COMM=$(cat /proc/$TARGET_PID/comm)
PYTHON_PATH=$(readlink -f /proc/$TARGET_PID/exe)
echo "Successfully found PID: $TARGET_PID"
echo "Monitoring command: '$TARGET_COMM' at $PYTHON_PATH"
echo "Starting eBPF loop... Press Ctrl+C to stop."

BPTRACE_CMD="uprobe:${PYTHON_PATH}:_PyEval_EvalFrameDefault /pid == $TARGET_PID/ { @calls = count(); } interval:s:1 { print(@calls); clear(@calls); }"

# --- Main Loop ---
stdbuf -oL bpftrace -e "$BPTRACE_CMD" | while read -r line; do
    
    calls=$(echo "$line" | awk -F '[: ]+' '{print $NF}')

    # --- THIS IS THE FIX ---
    # It now correctly checks for digits 0-9.
    if ! [[ "$calls" =~ ^[0-9]+$ ]]; then
        calls=0
    fi
    # --- END FIX ---

    clear
    
    # --- Bar Chart Logic ---
    BAR_MAX_WIDTH=50
    SCALE_MAX_CALLS=1000000
    
    if (( calls > 0 )); then
        percentage=$(echo "scale=4; $calls / $SCALE_MAX_CALLS" | bc)
        num_hashes=$(printf "%.0f" "$(echo "$percentage * $BAR_MAX_WIDTH" | bc)")
        
        if (( num_hashes == 0 )); then
            num_hashes=1
        fi
        if (( num_hashes > BAR_MAX_WIDTH )); then num_hashes=$BAR_MAX_WIDTH; fi
    else
        num_hashes=0
    fi
    
    bar=""
    for ((i=0; i<$num_hashes; i++)); do bar+="#"; done
    # --- End Bar Chart Logic ---

    # --- Draw the Dashboard ---
    echo "--- eBPF Live Python Interpreter Monitor ---"
    echo "Target PID: $TARGET_PID"
    echo "Press Ctrl+C to stop this monitor."
    echo
    echo "Interpreter Calls / sec ('_PyEval_EvalFrameDefault')"
    echo "----------------------------------------------------"
    printf "|%-*s| %d calls\n" $BAR_MAX_WIDTH "$bar" "$calls"
    echo "----------------------------------------------------"
    echo
    echo "To trigger high load, run this in another terminal:"
    echo "ab -n 100 -c 10 http://127.0.0.1:5000/slow"
done
