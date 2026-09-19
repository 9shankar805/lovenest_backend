package database

import (
	"os"
	"testing"
)

func TestInitDBAndTables(t *testing.T) {
	testDB := "test_lovenest.db"
	defer os.Remove(testDB)

	db, err := InitDB(testDB)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	tables := []string{"users", "couples", "scenes", "avatars", "messages", "checklists", "moods", "memories", "milestones"}
	for _, tbl := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
		if err != nil {
			t.Errorf("Expected table %s to exist, err: %v", tbl, err)
		}
	}
}
