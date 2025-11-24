# Real-Time eBPF Python Profiler

A lightweight, zero-instrumentation CPU profiler for Python applications. It uses bpftrace and FlameGraphs to visualize performance bottlenecks in real-time with negligible overhead (<1%).

## Quick Start

1. **Install Dependencies** You need a Linux environment with root access.

    ```
    sudo apt-get update
    sudo apt-get install -y bpftrace apache2-utils git bc
    ```

2. Clone the Repo
    ```
    git clone https://github.com/cnu1812/eBPF.git
    cd ebpf
    ```

3. Run the Demo Open 3 separate terminals:

- Terminal 1 (Target App):

    `python3 slow_app.py`

- Terminal 2 (Profiler):
    ```
    chmod +x realtime.sh
    ./realtime.sh
    ```

- Terminal 3 (Dashboard):

    `python3 -m http.server --directory ./www/html 8080`

4. View the Dashboard Open `http://localhost:8080` in your browser. Generate load in a 4th terminal to see the graph populate:

    `ab -n 100 -c 10 http://127.0.0.1:5000/slow` 

This tool works by sampling stack traces from the kernel, meaning no code changes are required in your application.

Probe: `profile:hz:99` (Samples 99 times/sec)

Filter: Filters by PID to isolate the target application.

Visualization: Uses Brendan Gregg's FlameGraph scripts.

If needed edit `realtime.sh` to adjust parameters:


## Troubleshooting

- "Permission Denied" / "Operation not permitted"

    eBPF requires root privileges. Ensure you are running `./realtime.sh` as a user with sudo access.

    If running in a container, you must run with `--privileged`.

- "No stacks captured"

    Idle App: The profiler only captures active CPU cycles. If the app is idle, the graph will be empty. Run ab to generate load.

    Missing Frame Pointers: If your Python binary was compiled without frame pointers (common in Alpine Linux), eBPF cannot walk the stack. Use a standard glibc-based Python (Debian/Ubuntu).

- Browser shows 404

    The profile.svg file is only generated after the first successful capture cycle (5 seconds). Wait for the script to say "Updated profile.svg".