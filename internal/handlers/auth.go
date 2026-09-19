package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/lovenest/backend/internal/database"
	"github.com/lovenest/backend/internal/middleware"
	"github.com/lovenest/backend/internal/models"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid request payload"})
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Name, email and password are required"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Password processing failed"})
		return
	}

	res, err := database.DB.Exec(`
		INSERT INTO users (email, password_hash, display_name)
		VALUES (?, ?, ?)
	`, req.Email, string(hashed), req.Name)

	if err != nil {
		respondJSON(w, http.StatusConflict, models.APIResponse{Success: false, Error: "Email already exists"})
		return
	}

	userID, _ := res.LastInsertId()
	token, err := middleware.GenerateToken(userID, nil)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Token generation failed"})
		return
	}

	user := models.User{
		ID:          userID,
		Email:       req.Email,
		DisplayName: req.Name,
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"token": token,
			"user":  user,
		},
	})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid request payload"})
		return
	}

	var user models.User
	var passwordHash string
	var coupleID, partnerID sql.NullInt64

	err := database.DB.QueryRow(`
		SELECT id, email, password_hash, display_name, photo_url, device_token, couple_id, partner_id, created_at, updated_at
		FROM users WHERE email = ?
	`, req.Email).Scan(
		&user.ID, &user.Email, &passwordHash, &user.DisplayName, &user.PhotoURL,
		&user.DeviceToken, &coupleID, &partnerID, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{Success: false, Error: "Invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		respondJSON(w, http.StatusUnauthorized, models.APIResponse{Success: false, Error: "Invalid email or password"})
		return
	}

	if coupleID.Valid {
		user.CoupleID = &coupleID.Int64
	}
	if partnerID.Valid {
		user.PartnerID = &partnerID.Int64
	}

	token, err := middleware.GenerateToken(user.ID, user.CoupleID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Token generation failed"})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"token": token,
			"user":  user,
		},
	})
}

func Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var user models.User
	var coupleID, partnerID sql.NullInt64

	err := database.DB.QueryRow(`
		SELECT id, email, display_name, photo_url, device_token, couple_id, partner_id, created_at, updated_at
		FROM users WHERE id = ?
	`, userID).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.PhotoURL,
		&user.DeviceToken, &coupleID, &partnerID, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		respondJSON(w, http.StatusNotFound, models.APIResponse{Success: false, Error: "User not found"})
		return
	}

	if coupleID.Valid {
		user.CoupleID = &coupleID.Int64
	}
	if partnerID.Valid {
		user.PartnerID = &partnerID.Int64
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"user": user,
		},
	})
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	coupleID := middleware.GetCoupleID(r.Context())

	token, err := middleware.GenerateToken(userID, coupleID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Token generation failed"})
		return
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"token": token,
		},
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
