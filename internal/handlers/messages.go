package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/lovenest/backend/internal/database"
	"github.com/lovenest/backend/internal/middleware"
	"github.com/lovenest/backend/internal/models"
)

var BroadcastChatMessage func(coupleID int64, senderID int64, message models.Message)

func GetMessages(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: []models.Message{}})
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, couple_id, sender_id, content, msg_type, reply_to_id, reply_preview, is_read, created_at
		FROM messages WHERE couple_id = ? ORDER BY id ASC LIMIT 100
	`, *coupleID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to fetch messages"})
		return
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		var replyID sql.NullInt64
		var isReadInt int
		if err := rows.Scan(&m.ID, &m.CoupleID, &m.SenderID, &m.Content, &m.Type, &replyID, &m.ReplyPreview, &isReadInt, &m.CreatedAt); err == nil {
			if replyID.Valid {
				m.ReplyToID = &replyID.Int64
			}
			m.IsRead = isReadInt == 1
			messages = append(messages, m)
		}
	}

	if messages == nil {
		messages = []models.Message{}
	}

	respondJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: messages})
}

type SendMessageRequest struct {
	Content      string `json:"content"`
	Type         string `json:"type"`
	ReplyToID    *int64 `json:"reply_to_id"`
	ReplyPreview string `json:"reply_preview"`
}

func SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID == nil || *coupleID == 0 {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "You must pair with your partner before sending messages"})
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid payload"})
		return
	}

	if req.Type == "" {
		req.Type = "text"
	}

	res, err := database.DB.Exec(`
		INSERT INTO messages (couple_id, sender_id, content, msg_type, reply_to_id, reply_preview)
		VALUES (?, ?, ?, ?, ?, ?)
	`, *coupleID, userID, req.Content, req.Type, req.ReplyToID, req.ReplyPreview)

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: "Failed to save message"})
		return
	}

	msgID, _ := res.LastInsertId()
	var msg models.Message
	var replyID sql.NullInt64
	var isReadInt int

	database.DB.QueryRow(`
		SELECT id, couple_id, sender_id, content, msg_type, reply_to_id, reply_preview, is_read, created_at
		FROM messages WHERE id = ?
	`, msgID).Scan(&msg.ID, &msg.CoupleID, &msg.SenderID, &msg.Content, &msg.Type, &replyID, &msg.ReplyPreview, &isReadInt, &msg.CreatedAt)

	if replyID.Valid {
		msg.ReplyToID = &replyID.Int64
	}
	msg.IsRead = isReadInt == 1

	if BroadcastChatMessage != nil {
		BroadcastChatMessage(*coupleID, userID, msg)
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    msg,
	})
}

func MarkRead(w http.ResponseWriter, r *http.Request) {
	coupleID := middleware.GetCoupleID(r.Context())
	if coupleID != nil && *coupleID > 0 {
		database.DB.Exec(`UPDATE messages SET is_read = 1 WHERE couple_id = ?`, *coupleID)
	}
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true})
}

func DeleteMessage(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid message ID"})
		return
	}
	msgID, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	database.DB.Exec(`DELETE FROM messages WHERE id = ?`, msgID)
	respondJSON(w, http.StatusOK, models.APIResponse{Success: true})
}
