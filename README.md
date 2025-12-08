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
    git submodule update --init --recursive
    cd ebpf
    ```

3. Run the Demo Open 3 separate terminals:

- Terminal 1 (Target App):

    ```
    python3 slow_app.py
    ```

    ![T1](https://github.com/user-attachments/assets/eeccb85b-b952-48dc-b0e0-b27fae4832be)

  

- Terminal 2 (Profiler):
    ```
    chmod +x realtime.sh
    ./realtime.sh
    ```

    ![T2](https://github.com/user-attachments/assets/b9d7f5c0-6c02-403d-beec-3e4fa1cf4db2)


- Terminal 3 (Dashboard):

    ```
    python3 -m http.server --directory ./www/html 8080
    ```

  ![T3](https://github.com/user-attachments/assets/97dbf968-ec05-4c8d-9775-759ff896c6a4)


4. View the Dashboard Open `http://localhost:8080` in your browser. Generate load in a 4th terminal to see the graph populate:

    ```
    ab -n 100 -c 10 http://127.0.0.1:5000/slow
    ```

   ![T4](https://github.com/user-attachments/assets/c027bfcf-7fb6-4df1-9a9e-e95efe631341)

   ## Result

   ![output](https://github.com/user-attachments/assets/1f2a396c-8913-4ecc-b8ea-a4fd072db1d1)

   


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
