package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lovenest/backend/internal/database"
	"github.com/lovenest/backend/internal/handlers"
	"github.com/lovenest/backend/internal/middleware"
	"github.com/lovenest/backend/internal/ws"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "lovenest.db"
	}

	// 1. Initialize SQLite Database
	if _, err := database.InitDB(dbPath); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	// 2. Initialize Real-Time WebSocket Hub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	go hub.Run()

	// 3. Router Setup
	mux := http.NewServeMux()

	// Health Check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"app":       "LoveNest Backend Server (Go)",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Real-Time WebSocket
	mux.HandleFunc("GET /api/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.HandleWebSocket(hub, w, r)
	})

	// Public Auth Endpoints
	mux.HandleFunc("POST /api/auth/register", handlers.Register)
	mux.HandleFunc("POST /api/auth/login", handlers.Login)

	// Protected Endpoints Submux
	protectedMux := http.NewServeMux()

	// Auth
	protectedMux.HandleFunc("GET /api/auth/me", handlers.Me)
	protectedMux.HandleFunc("POST /api/auth/refresh", handlers.Refresh)
	protectedMux.HandleFunc("POST /api/auth/logout", handlers.Logout)

	// Profile & Couple
	protectedMux.HandleFunc("GET /api/profile", handlers.GetProfile)
	protectedMux.HandleFunc("PATCH /api/profile", handlers.UpdateProfile)
	protectedMux.HandleFunc("GET /api/couple", handlers.GetCouple)
	protectedMux.HandleFunc("PATCH /api/couple", handlers.UpdateCouple)

	// Pairing
	protectedMux.HandleFunc("POST /api/pairing/generate-code", handlers.GeneratePairingCode)
	protectedMux.HandleFunc("POST /api/pairing/connect", handlers.ConnectPairing)

	// Scene of Life & Avatars
	protectedMux.HandleFunc("GET /api/scene", handlers.GetScene)
	protectedMux.HandleFunc("PUT /api/scene", handlers.SaveScene)
	protectedMux.HandleFunc("GET /api/avatar", handlers.GetAvatar)
	protectedMux.HandleFunc("PUT /api/avatar", handlers.SaveAvatar)
	protectedMux.HandleFunc("GET /api/avatar/partner", handlers.GetPartnerAvatar)

	// Messages (Chat)
	protectedMux.HandleFunc("GET /api/messages", handlers.GetMessages)
	protectedMux.HandleFunc("POST /api/messages", handlers.SendMessage)
	protectedMux.HandleFunc("POST /api/messages/mark-read", handlers.MarkRead)
	protectedMux.HandleFunc("DELETE /api/messages/", handlers.DeleteMessage)

	// Checklists
	protectedMux.HandleFunc("GET /api/checklist", handlers.GetChecklist)
	protectedMux.HandleFunc("POST /api/checklist", handlers.AddChecklistItem)
	protectedMux.HandleFunc("POST /api/checklist/", handlers.ToggleChecklistItem)
	protectedMux.HandleFunc("DELETE /api/checklist/", handlers.DeleteChecklistItem)

	// Moods
	protectedMux.HandleFunc("GET /api/moods", handlers.GetMoods)
	protectedMux.HandleFunc("GET /api/moods/today", handlers.GetTodayMoods)
	protectedMux.HandleFunc("POST /api/moods", handlers.LogMood)

	// Memories
	protectedMux.HandleFunc("GET /api/memories", handlers.GetMemories)
	protectedMux.HandleFunc("POST /api/memories", handlers.AddMemory)
	protectedMux.HandleFunc("DELETE /api/memories/", handlers.DeleteMemory)

	// Milestones
	protectedMux.HandleFunc("GET /api/milestones", handlers.GetMilestones)
	protectedMux.HandleFunc("POST /api/milestones", handlers.AddMilestone)
	protectedMux.HandleFunc("DELETE /api/milestones/", handlers.DeleteMilestone)

	// Mount protected routes under AuthMiddleware
	mux.Handle("/api/", middleware.AuthMiddleware(protectedMux))

	// Global Middleware: CORS & Logging
	handler := middleware.CORSMiddleware(requestLogger(mux))

	addr := fmt.Sprintf(":%s", port)
	log.Printf("💖 LoveNest Go Backend running at http://localhost%s (for Android emulator: http://10.0.2.2%s)", addr, addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s (%v)", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}
