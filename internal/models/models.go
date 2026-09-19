package models

import "time"

type User struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email"`
	Password    string     `json:"-"`
	DisplayName string     `json:"display_name"`
	PhotoURL    string     `json:"photo_url,omitempty"`
	DeviceToken string     `json:"device_token,omitempty"`
	CoupleID    *int64     `json:"couple_id,omitempty"`
	PartnerID   *int64     `json:"partner_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Couple struct {
	ID          int64      `json:"id"`
	PairingCode string     `json:"pairing_code"`
	CoupleName  string     `json:"couple_name"`
	StartDate   string     `json:"start_date"` // YYYY-MM-DD
	User1ID     int64      `json:"user1_id"`
	User2ID     *int64     `json:"user2_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SceneStateData struct {
	ThemeID   string                   `json:"theme_id,omitempty"`
	IsDayMode bool                     `json:"is_day_mode,omitempty"`
	PetID     string                   `json:"pet_id,omitempty"`
	Room      map[string]interface{}   `json:"room,omitempty"`
	Items     []map[string]interface{} `json:"items"`
	TimeOfDay string                   `json:"timeOfDay,omitempty"`
	Weather   string                   `json:"weather,omitempty"`
	UpdatedAt string                   `json:"updated_at,omitempty"`
}

type Message struct {
	ID           int64     `json:"id"`
	CoupleID     int64     `json:"couple_id"`
	SenderID     int64     `json:"sender_id"`
	Content      string    `json:"content"`
	Type         string    `json:"type"`
	ReplyToID    *int64    `json:"reply_to_id,omitempty"`
	ReplyPreview string    `json:"reply_preview,omitempty"`
	IsRead       bool      `json:"is_read"`
	CreatedAt    time.Time `json:"created_at"`
}

type ChecklistItem struct {
	ID        int64     `json:"id"`
	CoupleID  int64     `json:"couple_id"`
	Title     string    `json:"title"`
	Emoji     string    `json:"emoji"`
	Category  string    `json:"category"`
	Note      string    `json:"note,omitempty"`
	IsDone    bool      `json:"is_done"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type MoodEntry struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	CoupleID  int64     `json:"couple_id"`
	Emoji     string    `json:"emoji"`
	Label     string    `json:"label"`
	Intensity int       `json:"intensity"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type MemoryItem struct {
	ID          int64     `json:"id"`
	CoupleID    int64     `json:"couple_id"`
	Title       string    `json:"title"`
	Emoji       string    `json:"emoji"`
	Category    string    `json:"category"`
	Description string    `json:"description,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	HappenedAt  string    `json:"happened_at"` // YYYY-MM-DD
	CreatedAt   time.Time `json:"created_at"`
}

type MilestoneItem struct {
	ID          int64     `json:"id"`
	CoupleID    int64     `json:"couple_id"`
	Title       string    `json:"title"`
	Emoji       string    `json:"emoji"`
	Date        string    `json:"date"` // YYYY-MM-DD
	Type        string    `json:"type"`
	IsRecurring bool      `json:"is_recurring"`
	Note        string    `json:"note,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Standard API response structure matching Flutter ApiClient expectation
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
