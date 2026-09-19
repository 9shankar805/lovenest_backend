package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/lovenest/backend/internal/database"
	"github.com/lovenest/backend/internal/middleware"
	"github.com/lovenest/backend/internal/models"
)

func GeneratePairingCode(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Generate 4 random digits e.g. LOVE-4829
	b := make([]byte, 2)
	rand.Read(b)
	code := fmt.Sprintf("LOVE-%04d", (int(b[0])<<8|int(b[1]))%10000)

	// Check if user already has an active couple where partner is not yet paired
	var existingCoupleID int64
	err := database.DB.QueryRow(`
		SELECT id FROM couples WHERE user1_id = ? AND user2_id IS NULL
	`, userID).Scan(&existingCoupleID)

	if err == nil {
		// Update existing code
		database.DB.Exec(`UPDATE couples SET pairing_code = ? WHERE id = ?`, code, existingCoupleID)
	} else {
		// Create new couple record
		res, err := database.DB.Exec(`
			INSERT INTO couples (pairing_code, couple_name, user1_id)
			VALUES (?, 'Our Cozy Nest', ?)
		`, code, userID)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Could not create couple code"})
			return
		}
		coupleID, _ := res.LastInsertId()
		database.DB.Exec(`UPDATE users SET couple_id = ? WHERE id = ?`, coupleID, userID)
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"code": code,
		},
	})
}

type ConnectRequest struct {
	Code string `json:"code"`
}

func ConnectPairing(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	cleanCode := strings.ToUpper(strings.TrimSpace(req.Code))
	var couple models.Couple

	err := database.DB.QueryRow(`
		SELECT id, pairing_code, couple_name, start_date, user1_id
		FROM couples WHERE pairing_code = ? AND user2_id IS NULL
	`, cleanCode).Scan(&couple.ID, &couple.PairingCode, &couple.CoupleName, &couple.StartDate, &couple.User1ID)

	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusNotFound, models.APIResponse{Success: false, Error: "Invalid pairing code or already paired"})
		return
	} else if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Database error checking code"})
		return
	}

	if couple.User1ID == userID {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Cannot pair with yourself"})
		return
	}

	// Pair the two users!
	couple.User2ID = &userID
	_, err = database.DB.Exec(`
		UPDATE couples SET user2_id = ? WHERE id = ?
	`, userID, couple.ID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to pair couple"})
		return
	}

	// Update user2 (current user)
	database.DB.Exec(`UPDATE users SET couple_id = ?, partner_id = ? WHERE id = ?`, couple.ID, couple.User1ID, userID)
	// Update user1 (partner)
	database.DB.Exec(`UPDATE users SET couple_id = ?, partner_id = ? WHERE id = ?`, couple.ID, userID, couple.User1ID)

	// Initialize default starter Scene for the couple if not existing
	defaultRoomJSON := `{"id":"main_room","name":"Our Cozy Sanctuary","gridWidth":10,"gridHeight":10,"floorStyle":"oakParquet","wallStyle":"pastelRose","hasWindows":true,"hasDoor":true}`
	defaultItemsJSON := `[
		{"id":"bed_1","assetId":"furniture_bed_double","name":"Cozy Double Bed","category":"beds","x":1.0,"y":1.0,"width":2,"height":3,"rotation":0,"colorVariant":0,"layer":0},
		{"id":"sofa_1","assetId":"furniture_sofa_cozy","name":"Plush 2-Seater Sofa","category":"seating","x":5.0,"y":2.0,"width":2,"height":2,"rotation":0,"colorVariant":0,"layer":0},
		{"id":"rug_1","assetId":"rug_floral_oval","name":"Oval Floral Rug","category":"rugs","x":4.0,"y":4.0,"width":3,"height":3,"rotation":0,"colorVariant":0,"layer":-1},
		{"id":"lamp_1","assetId":"lamp_floor_arc","name":"Warm Floor Arc Lamp","category":"lighting","x":7.0,"y":2.0,"width":1,"height":1,"rotation":0,"colorVariant":0,"layer":1,"isLightOn":true},
		{"id":"plant_1","assetId":"plant_monstera","name":"Monstera Deliciosa","category":"plants","x":1.0,"y":5.0,"width":1,"height":1,"rotation":0,"colorVariant":0,"layer":1}
	]`

	database.DB.Exec(`
		INSERT OR IGNORE INTO scenes (couple_id, theme_id, room_json, items_json, time_of_day, weather)
		VALUES (?, 'cozy_studio', ?, ?, 'afternoon', 'sunny')
	`, couple.ID, defaultRoomJSON, defaultItemsJSON)

	// Fetch updated user
	var updatedUser models.User
	var coupleID, partnerID sql.NullInt64
	database.DB.QueryRow(`
		SELECT id, email, display_name, photo_url, couple_id, partner_id FROM users WHERE id = ?
	`, userID).Scan(&updatedUser.ID, &updatedUser.Email, &updatedUser.DisplayName, &updatedUser.PhotoURL, &coupleID, &partnerID)

	if coupleID.Valid {
		updatedUser.CoupleID = &coupleID.Int64
	}
	if partnerID.Valid {
		updatedUser.PartnerID = &partnerID.Int64
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"couple": couple,
			"user":   updatedUser,
		},
	})
}
