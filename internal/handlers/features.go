package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lovenest/backend/internal/database"
	"github.com/lovenest/backend/internal/middleware"
	"github.com/lovenest/backend/internal/models"
)

// ── Checklists ───────────────────────────────────────────────────────────────

func GetChecklist(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: []models.ChecklistItem{}})
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, couple_id, title, emoji, category, note, is_done, created_by, created_at
		FROM checklists WHERE couple_id = ? ORDER BY id DESC
	`, *coupleID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Database error"})
		return
	}
	defer rows.Close()

	var list []models.ChecklistItem
	for rows.Next() {
		var item models.ChecklistItem
		var isDoneInt int
		if err := rows.Scan(&item.ID, &item.CoupleID, &item.Title, &item.Emoji, &item.Category, &item.Note, &isDoneInt, &item.CreatedBy, &item.CreatedAt); err == nil {
			item.IsDone = isDoneInt == 1
			list = append(list, item)
		}
	}
	if list == nil {
		list = []models.ChecklistItem{}
	}
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: list})
}

type AddChecklistRequest struct {
	Title    string `json:"title"`
	Emoji    string `json:"emoji"`
	Category string `json:"category"`
	Note     string `json:"note"`
}

func AddChecklistItem(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Must be in a couple"})
		return
	}

	var req AddChecklistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	if req.Emoji == "" {
		req.Emoji = "📝"
	}
	if req.Category == "" {
		req.Category = "general"
	}

	res, err := database.DB.Exec(`
		INSERT INTO checklists (couple_id, title, emoji, category, note, created_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`, *coupleID, req.Title, req.Emoji, req.Category, req.Note, userID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to save checklist item"})
		return
	}

	id, _ := res.LastInsertId()
	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":         id,
			"couple_id":  *coupleID,
			"title":      req.Title,
			"emoji":      req.Emoji,
			"category":   req.Category,
			"note":       req.Note,
			"is_done":    false,
			"created_by": userID,
		},
	})
}

func ToggleChecklistItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	// e.g. /api/checklist/5/toggle
	var id int64
	for i, p := range parts {
		if p == "checklist" && i+1 < len(parts) {
			id, _ = strconv.ParseInt(parts[i+1], 10, 64)
			break
		}
	}

	database.DB.Exec(`UPDATE checklists SET is_done = CASE WHEN is_done = 1 THEN 0 ELSE 1 END WHERE id = ?`, id)
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true})
}

func DeleteChecklistItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	database.DB.Exec(`DELETE FROM checklists WHERE id = ?`, id)
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true})
}

// ── Moods ─────────────────────────────────────────────────────────────────────

func GetMoods(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: []models.MoodEntry{}})
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, user_id, couple_id, emoji, label, intensity, note, created_at
		FROM moods WHERE couple_id = ? ORDER BY id DESC LIMIT 50
	`, *coupleID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Database error"})
		return
	}
	defer rows.Close()

	var list []models.MoodEntry
	for rows.Next() {
		var m models.MoodEntry
		if err := rows.Scan(&m.ID, &m.UserID, &m.CoupleID, &m.Emoji, &m.Label, &m.Intensity, &m.Note, &m.CreatedAt); err == nil {
			list = append(list, m)
		}
	}
	if list == nil {
		list = []models.MoodEntry{}
	}
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: list})
}

func GetTodayMoods(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]interface{}{"mine": nil, "partner": nil}})
		return
	}

	userID := middleware.GetUserID(r.Context())
	today := time.Now().Format("2006-01-02")

	rows, _ := database.DB.Query(`
		SELECT id, user_id, couple_id, emoji, label, intensity, note, created_at
		FROM moods WHERE couple_id = ? AND date(created_at) = ? ORDER BY id DESC
	`, *coupleID, today)
	defer rows.Close()

	var mine, partner *models.MoodEntry
	for rows.Next() {
		var m models.MoodEntry
		if err := rows.Scan(&m.ID, &m.UserID, &m.CoupleID, &m.Emoji, &m.Label, &m.Intensity, &m.Note, &m.CreatedAt); err == nil {
			if m.UserID == userID && mine == nil {
				mine = &m
			} else if m.UserID != userID && partner == nil {
				partner = &m
			}
		}
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"mine":    mine,
			"partner": partner,
		},
	})
}

type LogMoodRequest struct {
	Emoji     string `json:"emoji"`
	Label     string `json:"label"`
	Intensity int    `json:"intensity"`
	Note      string `json:"note"`
}

func LogMood(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	coupleID := middleware.GetCoupleID(r.Context())
	cid := int64(0)
	if coupleID != nil {
		cid = *coupleID
	}

	var req LogMoodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	res, err := database.DB.Exec(`
		INSERT INTO moods (user_id, couple_id, emoji, label, intensity, note)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, cid, req.Emoji, req.Label, req.Intensity, req.Note)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to log mood"})
		return
	}

	id, _ := res.LastInsertId()
	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":        id,
			"user_id":   userID,
			"emoji":     req.Emoji,
			"label":     req.Label,
			"intensity": req.Intensity,
			"note":      req.Note,
		},
	})
}

// ── Memories ──────────────────────────────────────────────────────────────────

func GetMemories(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: []models.MemoryItem{}})
		return
	}

	category := r.URL.Query().Get("category")
	var rows *sql.Rows
	var err error

	if category != "" {
		rows, err = database.DB.Query(`
			SELECT id, couple_id, title, emoji, category, description, image_url, tags_json, happened_at, created_at
			FROM memories WHERE couple_id = ? AND category = ? ORDER BY happened_at DESC
		`, *coupleID, category)
	} else {
		rows, err = database.DB.Query(`
			SELECT id, couple_id, title, emoji, category, description, image_url, tags_json, happened_at, created_at
			FROM memories WHERE couple_id = ? ORDER BY happened_at DESC
		`, *coupleID)
	}

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Database error"})
		return
	}
	defer rows.Close()

	var list []models.MemoryItem
	for rows.Next() {
		var m models.MemoryItem
		var tagsJSON string
		if err := rows.Scan(&m.ID, &m.CoupleID, &m.Title, &m.Emoji, &m.Category, &m.Description, &m.ImageURL, &tagsJSON, &m.HappenedAt, &m.CreatedAt); err == nil {
			json.Unmarshal([]byte(tagsJSON), &m.Tags)
			list = append(list, m)
		}
	}
	if list == nil {
		list = []models.MemoryItem{}
	}
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: list})
}

type AddMemoryRequest struct {
	Title       string   `json:"title"`
	Emoji       string   `json:"emoji"`
	HappenedAt  string   `json:"happened_at"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	Tags        []string `json:"tags"`
}

func AddMemory(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Must be in a couple"})
		return
	}

	var req AddMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	tagsBytes, _ := json.Marshal(req.Tags)
	res, err := database.DB.Exec(`
		INSERT INTO memories (couple_id, title, emoji, category, description, image_url, tags_json, happened_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, *coupleID, req.Title, req.Emoji, req.Category, req.Description, req.ImageURL, string(tagsBytes), req.HappenedAt)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to create memory"})
		return
	}

	id, _ := res.LastInsertId()
	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":          id,
			"title":       req.Title,
			"emoji":       req.Emoji,
			"happened_at": req.HappenedAt,
			"category":    req.Category,
		},
	})
}

func DeleteMemory(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	database.DB.Exec(`DELETE FROM memories WHERE id = ?`, id)
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true})
}

// ── Milestones ────────────────────────────────────────────────────────────────

func GetMilestones(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: []models.MilestoneItem{}})
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, couple_id, title, emoji, date, type, is_recurring, note, created_at
		FROM milestones WHERE couple_id = ? ORDER BY date ASC
	`, *coupleID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Database error"})
		return
	}
	defer rows.Close()

	var list []models.MilestoneItem
	for rows.Next() {
		var m models.MilestoneItem
		var isRecInt int
		if err := rows.Scan(&m.ID, &m.CoupleID, &m.Title, &m.Emoji, &m.Date, &m.Type, &isRecInt, &m.Note, &m.CreatedAt); err == nil {
			m.IsRecurring = isRecInt == 1
			list = append(list, m)
		}
	}
	if list == nil {
		list = []models.MilestoneItem{}
	}
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: list})
}

type AddMilestoneRequest struct {
	Title       string `json:"title"`
	Emoji       string `json:"emoji"`
	Date        string `json:"date"`
	Type        string `json:"type"`
	IsRecurring bool   `json:"is_recurring"`
	Note        string `json:"note"`
}

func AddMilestone(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Must be in a couple"})
		return
	}

	var req AddMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	isRecInt := 0
	if req.IsRecurring {
		isRecInt = 1
	}

	res, err := database.DB.Exec(`
		INSERT INTO milestones (couple_id, title, emoji, date, type, is_recurring, note)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, *coupleID, req.Title, req.Emoji, req.Date, req.Type, isRecInt, req.Note)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to add milestone"})
		return
	}

	id, _ := res.LastInsertId()
	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":           id,
			"title":        req.Title,
			"emoji":        req.Emoji,
			"date":         req.Date,
			"type":         req.Type,
			"is_recurring": req.IsRecurring,
		},
	})
}

func DeleteMilestone(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	database.DB.Exec(`DELETE FROM milestones WHERE id = ?`, id)
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true})
}
