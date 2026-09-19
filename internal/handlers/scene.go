package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/lovenest/backend/internal/database"
	"github.com/lovenest/backend/internal/middleware"
	"github.com/lovenest/backend/internal/models"
)

// Global event broadcaster func hook set by ws package
var BroadcastSceneEvent func(coupleID int64, senderID int64, eventType string, payload interface{})

func GetScene(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	coupleID := middleware.GetCoupleID(r.Context())

	targetID := int64(0)
	if coupleID != nil && *coupleID > 0 {
		targetID = *coupleID
	} else {
		// Use user ID as single scene fallback
		targetID = -userID
	}

	var themeID, petID, roomJSON, itemsJSON, timeOfDay, weather string
	var isDayMode int
	var updatedAt time.Time

	err := database.DB.QueryRow(`
		SELECT theme_id, is_day_mode, pet_id, room_json, items_json, time_of_day, weather, updated_at
		FROM scenes WHERE couple_id = ?
	`, targetID).Scan(&themeID, &isDayMode, &petID, &roomJSON, &itemsJSON, &timeOfDay, &weather, &updatedAt)

	if err == sql.ErrNoRows {
		// Return empty starter default
		defaultRoom := map[string]interface{}{
			"id": "main_room", "name": "Our Cozy Sanctuary", "gridWidth": 10, "gridHeight": 10,
			"floorStyle": "oakParquet", "wallStyle": "pastelRose", "hasWindows": true, "hasDoor": true,
		}
		respondJSON(w, http.StatusOK, models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"theme_id":    "cozy_studio",
				"is_day_mode": true,
				"pet_id":      "pet_1",
				"room":        defaultRoom,
				"items":       []interface{}{},
				"timeOfDay":   "afternoon",
				"weather":     "sunny",
			},
		})
		return
	} else if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Database error retrieving scene"})
		return
	}

	var roomMap map[string]interface{}
	var itemsList []map[string]interface{}
	json.Unmarshal([]byte(roomJSON), &roomMap)
	json.Unmarshal([]byte(itemsJSON), &itemsList)

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"theme_id":    themeID,
			"is_day_mode": isDayMode == 1,
			"pet_id":      petID,
			"room":        roomMap,
			"items":       itemsList,
			"timeOfDay":   timeOfDay,
			"weather":     weather,
			"updated_at":  updatedAt.Format(time.RFC3339),
		},
	})
}

type SaveSceneRequest struct {
	ThemeID   string                   `json:"theme_id"`
	IsDayMode bool                     `json:"is_day_mode"`
	PetID     string                   `json:"pet_id"`
	Room      map[string]interface{}   `json:"room"`
	Items     []map[string]interface{} `json:"items"`
	TimeOfDay string                   `json:"timeOfDay"`
	Weather   string                   `json:"weather"`
}

func SaveScene(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	coupleID := middleware.GetCoupleID(r.Context())

	targetID := int64(0)
	if coupleID != nil && *coupleID > 0 {
		targetID = *coupleID
	} else {
		targetID = -userID
	}

	var req SaveSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	roomBytes, _ := json.Marshal(req.Room)
	itemsBytes, _ := json.Marshal(req.Items)
	isDayInt := 0
	if req.IsDayMode {
		isDayInt = 1
	}

	_, err := database.DB.Exec(`
		INSERT INTO scenes (couple_id, theme_id, is_day_mode, pet_id, room_json, items_json, time_of_day, weather, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(couple_id) DO UPDATE SET
			theme_id = excluded.theme_id,
			is_day_mode = excluded.is_day_mode,
			pet_id = excluded.pet_id,
			room_json = CASE WHEN excluded.room_json != '{}' AND excluded.room_json != 'null' THEN excluded.room_json ELSE room_json END,
			items_json = excluded.items_json,
			time_of_day = CASE WHEN excluded.time_of_day != '' THEN excluded.time_of_day ELSE time_of_day END,
			weather = CASE WHEN excluded.weather != '' THEN excluded.weather ELSE weather END,
			updated_at = CURRENT_TIMESTAMP
	`, targetID, req.ThemeID, isDayInt, req.PetID, string(roomBytes), string(itemsBytes), req.TimeOfDay, req.Weather)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to persist scene: " + err.Error()})
		return
	}

	// Real-time broadcast to partner!
	if BroadcastSceneEvent != nil && coupleID != nil && *coupleID > 0 {
		BroadcastSceneEvent(*coupleID, userID, "scene:update", req)
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]interface{}{"saved": true},
	})
}

func GetAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var avatarJSON string

	err := database.DB.QueryRow(`SELECT avatar_json FROM avatars WHERE user_id = ?`, userID).Scan(&avatarJSON)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusOK, models.APIResponse{
			Success: true,
			Data:    map[string]interface{}{},
		})
		return
	}

	var data map[string]interface{}
	json.Unmarshal([]byte(avatarJSON), &data)
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: data})
}

func SaveAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid avatar payload"})
		return
	}

	b, _ := json.Marshal(body)
	_, err := database.DB.Exec(`
		INSERT INTO avatars (user_id, avatar_json, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET avatar_json = excluded.avatar_json, updated_at = CURRENT_TIMESTAMP
	`, userID, string(b))

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to save avatar"})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: body})
}

func GetPartnerAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var partnerID sql.NullInt64
	err := database.DB.QueryRow(`SELECT partner_id FROM users WHERE id = ?`, userID).Scan(&partnerID)
	if err != nil || !partnerID.Valid {
		respondJSON(w, http.StatusNotFound, models.APIResponse{Success: false, Error: "No partner found"})
		return
	}

	var avatarJSON string
	err = database.DB.QueryRow(`SELECT avatar_json FROM avatars WHERE user_id = ?`, partnerID.Int64).Scan(&avatarJSON)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]interface{}{}})
		return
	}

	var data map[string]interface{}
	json.Unmarshal([]byte(avatarJSON), &data)
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: data})
}
