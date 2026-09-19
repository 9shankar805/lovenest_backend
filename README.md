# lovenest_backend

High-performance, lightweight Go backend for **LoveNest** with pure-Go SQLite persistence, REST endpoints, and a real-time WebSocket hub for instant couple synchronization.

---

## 🚀 Features

- **Zero External C Compiler Dependency**: Powered by `modernc.org/sqlite` (pure Go). Runs out of the box on Windows, Linux, and macOS without GCC/MinGW.
- **Couple Pairing & Handshake**: Unique 6-character pairing codes (`LOVE-XXXX`) with partner binding.
- **Authentication & JWT**: Fast bcrypt password hashing and 30-day HS256 JWT tokens.
- **Scene of Life Engine**: Real-time room layout persistence, placed isometric furniture sync, Day/Night cycle, and weather states.
- **Real-Time WebSocket Hub (`/api/ws`)**:
  - Live partner avatar walking & path waypoints synchronization.
  - Live furniture placement, movement, and deletion sync.
  - Partner presence (`online` / `offline`).
  - Real-time couple chat messaging.
- **Complete Suite of Couple APIs**: Messages, Checklists, Mood logs, Memories with photo URLs & tags, and Milestones.

---

## 🛠️ Architecture

```
backend/
├── cmd/
│   └── server/
│       └── main.go         # HTTP Server entry point & route definitions
├── internal/
│   ├── database/
│   │   ├── db.go           # SQLite connection & automated schema migrations
│   │   └── db_test.go      # Unit tests
│   ├── handlers/
│   │   ├── auth.go         # Register, Login, Me, Refresh, Logout
│   │   ├── pairing.go      # GenerateCode, Connect
│   │   ├── couple.go       # Couple info, User profile
│   │   ├── scene.go        # Scene of life & Avatars
│   │   ├── messages.go     # Couple chat messages
│   │   └── features.go     # Checklists, Moods, Memories, Milestones
│   ├── middleware/
│   │   ├── auth.go         # JWT Bearer validation
│   │   └── cors.go         # CORS headers for mobile & web clients
│   ├── models/
│   │   └── models.go       # Structs & JSON models
│   └── ws/
│       └── hub.go          # Couple room WebSocket hub & client pump
├── go.mod
└── go.sum
```

---

## 📦 How to Run

### Requirements
- Go 1.22+ or Go 1.23+

### Run Locally
```bash
# Clone the repository
git clone https://github.com/9shankar805/lovenest_backend.git
cd lovenest_backend

# Run the server on port 8000
go run ./cmd/server
```

### Build Binary
```bash
go build -o server.exe ./cmd/server
./server.exe
```

The server runs on `http://localhost:8000` (and `http://10.0.2.2:8000` for Android emulator).

---

## 🧪 Tests
```bash
go test -v ./...
```
