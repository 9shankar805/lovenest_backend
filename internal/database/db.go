package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		dbPath = "lovenest.db"
	}

	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Enable WAL mode & busy timeout for high concurrency
	if _, err := db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA busy_timeout=5000;
		PRAGMA synchronous=NORMAL;
		PRAGMA foreign_keys=ON;
	`); err != nil {
		log.Printf("Warning setting PRAGMAs: %v", err)
	}

	DB = db
	if err := migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	log.Println("SQLite database initialized successfully at", dbPath)
	return DB, nil
}

func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		display_name TEXT NOT NULL,
		photo_url TEXT DEFAULT '',
		device_token TEXT DEFAULT '',
		couple_id INTEGER,
		partner_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS couples (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pairing_code TEXT UNIQUE NOT NULL,
		couple_name TEXT NOT NULL DEFAULT 'Us Two',
		start_date TEXT DEFAULT '',
		user1_id INTEGER NOT NULL,
		user2_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS scenes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		couple_id INTEGER UNIQUE NOT NULL,
		theme_id TEXT DEFAULT 'cozy_studio',
		is_day_mode INTEGER DEFAULT 1,
		pet_id TEXT DEFAULT 'pet_1',
		room_json TEXT NOT NULL DEFAULT '{}',
		items_json TEXT NOT NULL DEFAULT '[]',
		time_of_day TEXT DEFAULT 'afternoon',
		weather TEXT DEFAULT 'sunny',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS avatars (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER UNIQUE NOT NULL,
		avatar_json TEXT NOT NULL DEFAULT '{}',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		couple_id INTEGER NOT NULL,
		sender_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		msg_type TEXT NOT NULL DEFAULT 'text',
		reply_to_id INTEGER,
		reply_preview TEXT DEFAULT '',
		is_read INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS checklists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		couple_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		emoji TEXT NOT NULL DEFAULT '📝',
		category TEXT NOT NULL DEFAULT 'general',
		note TEXT DEFAULT '',
		is_done INTEGER DEFAULT 0,
		created_by INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS moods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		couple_id INTEGER NOT NULL,
		emoji TEXT NOT NULL,
		label TEXT NOT NULL,
		intensity INTEGER NOT NULL DEFAULT 3,
		note TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		couple_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		emoji TEXT NOT NULL DEFAULT '✨',
		category TEXT NOT NULL DEFAULT 'milestone',
		description TEXT DEFAULT '',
		image_url TEXT DEFAULT '',
		tags_json TEXT DEFAULT '[]',
		happened_at TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS milestones (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		couple_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		emoji TEXT NOT NULL DEFAULT '🎉',
		date TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'anniversary',
		is_recurring INTEGER DEFAULT 0,
		note TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_couple ON messages(couple_id, id DESC);
	CREATE INDEX IF NOT EXISTS idx_checklists_couple ON checklists(couple_id);
	CREATE INDEX IF NOT EXISTS idx_moods_couple ON moods(couple_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_memories_couple ON memories(couple_id, happened_at DESC);
	CREATE INDEX IF NOT EXISTS idx_milestones_couple ON milestones(couple_id, date ASC);
	`

	_, err := DB.Exec(schema)
	return err
}
