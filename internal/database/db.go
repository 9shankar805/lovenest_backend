package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type DBWrapper struct {
	*sql.DB
}

func (w *DBWrapper) QueryRow(query string, args ...interface{}) *sql.Row {
	return w.DB.QueryRow(Rebind(query), args...)
}

func (w *DBWrapper) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return w.DB.Query(Rebind(query), args...)
}

func (w *DBWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	return w.DB.Exec(Rebind(query), args...)
}

var DB *DBWrapper
var DriverName = "sqlite"

func InitDB(dbPath string) (*DBWrapper, error) {
	// Check if cloud managed PostgreSQL DATABASE_URL is provided (e.g. on DockHosting)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("POSTGRES_URL")
	}

	if dbURL != "" && (strings.HasPrefix(dbURL, "postgres://") || strings.HasPrefix(dbURL, "postgresql://")) {
		DriverName = "postgres"
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to open postgres database: %w", err)
		}
		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("failed to connect to postgres: %w", err)
		}
		DB = &DBWrapper{db}
		if err := migratePostgres(); err != nil {
			return nil, fmt.Errorf("postgres migration failed: %w", err)
		}
		log.Println("🐘 Managed PostgreSQL database connected and migrated successfully!")
		return DB, nil
	}

	// Fallback to embedded SQLite (for local offline dev)
	DriverName = "sqlite"
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

	if _, err := db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA busy_timeout=5000;
		PRAGMA synchronous=NORMAL;
		PRAGMA foreign_keys=ON;
	`); err != nil {
		log.Printf("Warning setting PRAGMAs: %v", err)
	}

	DB = &DBWrapper{db}
	if err := migrateSQLite(); err != nil {
		return nil, fmt.Errorf("sqlite migration failed: %w", err)
	}

	log.Println("🪶 SQLite database initialized successfully at", dbPath)
	return DB, nil
}

// Rebind converts standard '?' placeholders to PostgreSQL '$1, $2, ...' placeholders
func Rebind(query string) string {
	if DriverName != "postgres" {
		return query
	}
	var sb strings.Builder
	idx := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			sb.WriteString(fmt.Sprintf("$%d", idx))
			idx++
		} else {
			sb.WriteByte(query[i])
		}
	}
	return sb.String()
}

func Exec(query string, args ...interface{}) (sql.Result, error) {
	return DB.Exec(Rebind(query), args...)
}

func Query(query string, args ...interface{}) (*sql.Rows, error) {
	return DB.Query(Rebind(query), args...)
}

func QueryRow(query string, args ...interface{}) *sql.Row {
	return DB.QueryRow(Rebind(query), args...)
}

func migrateSQLite() error {
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

func migratePostgres() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		display_name TEXT NOT NULL,
		photo_url TEXT DEFAULT '',
		device_token TEXT DEFAULT '',
		couple_id BIGINT,
		partner_id BIGINT,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS couples (
		id BIGSERIAL PRIMARY KEY,
		pairing_code TEXT UNIQUE NOT NULL,
		couple_name TEXT NOT NULL DEFAULT 'Us Two',
		start_date TEXT DEFAULT '',
		user1_id BIGINT NOT NULL,
		user2_id BIGINT,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS scenes (
		id BIGSERIAL PRIMARY KEY,
		couple_id BIGINT UNIQUE NOT NULL,
		theme_id TEXT DEFAULT 'cozy_studio',
		is_day_mode INT DEFAULT 1,
		pet_id TEXT DEFAULT 'pet_1',
		room_json TEXT NOT NULL DEFAULT '{}',
		items_json TEXT NOT NULL DEFAULT '[]',
		time_of_day TEXT DEFAULT 'afternoon',
		weather TEXT DEFAULT 'sunny',
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS avatars (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT UNIQUE NOT NULL,
		avatar_json TEXT NOT NULL DEFAULT '{}',
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id BIGSERIAL PRIMARY KEY,
		couple_id BIGINT NOT NULL,
		sender_id BIGINT NOT NULL,
		content TEXT NOT NULL,
		msg_type TEXT NOT NULL DEFAULT 'text',
		reply_to_id BIGINT,
		reply_preview TEXT DEFAULT '',
		is_read INT DEFAULT 0,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS checklists (
		id BIGSERIAL PRIMARY KEY,
		couple_id BIGINT NOT NULL,
		title TEXT NOT NULL,
		emoji TEXT NOT NULL DEFAULT '📝',
		category TEXT NOT NULL DEFAULT 'general',
		note TEXT DEFAULT '',
		is_done INT DEFAULT 0,
		created_by BIGINT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS moods (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		couple_id BIGINT NOT NULL,
		emoji TEXT NOT NULL,
		label TEXT NOT NULL,
		intensity INT NOT NULL DEFAULT 3,
		note TEXT DEFAULT '',
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS memories (
		id BIGSERIAL PRIMARY KEY,
		couple_id BIGINT NOT NULL,
		title TEXT NOT NULL,
		emoji TEXT NOT NULL DEFAULT '✨',
		category TEXT NOT NULL DEFAULT 'milestone',
		description TEXT DEFAULT '',
		image_url TEXT DEFAULT '',
		tags_json TEXT DEFAULT '[]',
		happened_at TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS milestones (
		id BIGSERIAL PRIMARY KEY,
		couple_id BIGINT NOT NULL,
		title TEXT NOT NULL,
		emoji TEXT NOT NULL DEFAULT '🎉',
		date TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'anniversary',
		is_recurring INT DEFAULT 0,
		note TEXT DEFAULT '',
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_couple_pg ON messages(couple_id, id DESC);
	CREATE INDEX IF NOT EXISTS idx_checklists_couple_pg ON checklists(couple_id);
	CREATE INDEX IF NOT EXISTS idx_moods_couple_pg ON moods(couple_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_memories_couple_pg ON memories(couple_id, happened_at DESC);
	CREATE INDEX IF NOT EXISTS idx_milestones_couple_pg ON milestones(couple_id, date ASC);
	`

	_, err := DB.Exec(schema)
	return err
}
