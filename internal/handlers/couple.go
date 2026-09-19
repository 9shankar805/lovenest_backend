package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/lovenest/backend/internal/database"
	"github.com/lovenest/backend/internal/middleware"
	"github.com/lovenest/backend/internal/models"
)

func GetCouple(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusOK, models.APIResponse{
			Success: false,
			Error:   "Not paired with a partner yet",
		})
		return
	}

	var couple models.Couple
	var startDate sql.NullString
	var user2ID sql.NullInt64

	err := database.DB.QueryRow(`
		SELECT id, pairing_code, couple_name, start_date, user1_id, user2_id, created_at
		FROM couples WHERE id = ?
	`, *coupleID).Scan(&couple.ID, &couple.PairingCode, &couple.CoupleName, &startDate, &couple.User1ID, &user2ID, &couple.CreatedAt)

	if err != nil {
		respondJSON(w, http.StatusNotFound, models.APIResponse{Success: false, Error: "Couple not found"})
		return
	}

	if startDate.Valid {
		couple.StartDate = startDate.String
	}
	if user2ID.Valid {
		couple.User2ID = &user2ID.Int64
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    couple,
	})
}

type UpdateCoupleRequest struct {
	CoupleName string `json:"couple_name"`
	StartDate  string `json:"start_date"`
}

func UpdateCouple(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Not in a couple"})
		return
	}

	var req UpdateCoupleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	if req.CoupleName != "" {
		database.DB.Exec(`UPDATE couples SET couple_name = ? WHERE id = ?`, req.CoupleName, *coupleID)
	}
	if req.StartDate != "" {
		database.DB.Exec(`UPDATE couples SET start_date = ? WHERE id = ?`, req.StartDate, *coupleID)
	}

	GetCouple(w, r)
}

func GetProfile(w http.ResponseWriter, r *http.Request) {
	Me(w, r)
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	PhotoURL    string `json:"photo_url"`
	DeviceToken string `json:"device_token"`
}

func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	if req.DisplayName != "" {
		database.DB.Exec(`UPDATE users SET display_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, req.DisplayName, userID)
	}
	if req.PhotoURL != "" {
		database.DB.Exec(`UPDATE users SET photo_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, req.PhotoURL, userID)
	}
	if req.DeviceToken != "" {
		database.DB.Exec(`UPDATE users SET device_token = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, req.DeviceToken, userID)
	}

	Me(w, r)
}
