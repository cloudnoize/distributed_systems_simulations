# Gossip Simulation in Go

This project implements a simple **gossip-based dissemination protocol** in Go.  
Multiple HTTP servers (nodes) are started on sequential ports, and they exchange *version numbers* using a `/gossip` endpoint. Each node maintains a local record of the versions it has seen, and once all nodes have received a given version, the system reports it as **committed**.

---

## ⚙️ How it Works

- **Servers / Nodes**  
  - The program spawns `NUM_GOROUTINES` HTTP servers.  
  - Each server listens on `START_PORT + id`.  
  - Each server has its own local state `map[int]bool` to track which version numbers it has seen.

- **Gossip Protocol**  
  - Versions are integers (`1, 2, 3, …`) sent as the payload of an HTTP `PUT` to `/gossip`.  
  - When a server receives a *new* version:  
    1. It records it in its local state.  
    2. It forwards (gossips) the version to a random selection of peers.  
    3. It pushes the version into a **shared channel** so that the global `counterLoop` can track progress.  
  - If a server already saw the version, it ignores the request.

- **Commit Tracking**  
  - The `counterLoop` goroutine maintains a global `map[int]int` of how many servers have seen each version.  
  - When the count for a version equals the total number of nodes, that version is reported as **committed**.

- **Client Side (initiator)**  
  - A ticker runs every `SEND_INTERVAL` seconds, increments the version number, and sends it to random nodes via `gossipClient`.  
  - `gossipClient` picks `NUM_NODES_TO_SEND` random ports in the cluster and issues `PUT` requests.

---

## 🖥️ Environment Variables

Before running, configure the system via environment variables:

- `NUM_GOROUTINES` – number of nodes (servers) to start.  
- `START_PORT` – base port; nodes listen on sequential ports starting here.  
- `NUM_NODES_TO_SEND` – how many peers each gossip message is sent to.  
- `SEND_INTERVAL` – seconds between generating new versions at the source.

---

## ▶️ Running the Simulation

1. Initialize a Go module and run:
   ```bash
   go mod init gossip-sim
   go run .
   ```
2. Set environment variables and launch:
   ```bash
   export NUM_GOROUTINES=5
   export START_PORT=8000
   export NUM_NODES_TO_SEND=3
   export SEND_INTERVAL=10
   go run .
   ```

3. Observe logs:
   - Servers start on ports `8000 … 8005`.  
   - Every 10 seconds a new version is created and gossiped.  
   - Logs show versions spreading until they reach all nodes, at which point the version is committed.

---

## 🔍 Example Log Output

```
2025/10/02 Starting server 1 on port 8001
2025/10/02 Starting server 2 on port 8002
...
2025/10/02 Sending version 1
Received for 1 → count = 1
Received for 1 → count = 2
Committed version 1 → count = 6
```

---

## 🌱 Extensions

- Add **failure simulation**: randomly drop messages or kill nodes.  
- Add **message delay** to mimic network latency.  
- Extend gossip to carry richer state (key/value pairs, configs).  
- Visualize propagation using a simple dashboard.
