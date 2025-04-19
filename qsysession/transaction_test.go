package qsysession

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"qsyorm/qsydialect"
	"qsyorm/qsylog"

	_ "github.com/mattn/go-sqlite3"
)

func TestTransaction(t *testing.T) {
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		t.Fatal("failed to connect database", err)
	}
	defer func() {
		_ = db.Close()
		_ = os.Remove("test.db")
	}()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS User(Name text);")
	if err != nil {
		t.Fatal("failed to create table", err)
	}

	logger := qsylog.New(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile), qsylog.Config{
		Colorful: true,
		Loglevel: qsylog.Info,
	})
	dialect, _ := qsydialect.GetDialect("sqlite3")
	s := NewSession(db, logger, dialect)

	// Test Begin/Commit
	t.Run("TestBeginCommit", func(t *testing.T) {
		if err := s.Begin(); err != nil {
			t.Fatal("failed to begin transaction", err)
		}

		_, err = s.Raw("INSERT INTO User(Name) VALUES (?)", "Tom").Exec()
		if err != nil {
			t.Fatal("failed to insert", err)
		}

		if err := s.Commit(); err != nil {
			t.Fatal("failed to commit", err)
		}

		var name string
		row := s.Raw("SELECT Name FROM User LIMIT 1").QueryRow()
		if err := row.Scan(&name); err != nil {
			t.Fatal("failed to query", err)
		}

		if name != "Tom" {
			t.Fatalf("expected %s, got %s", "Tom", name)
		}
	})

	// Test Begin/Rollback
	t.Run("TestBeginRollback", func(t *testing.T) {
		count := countUsers(t, s)

		if err := s.Begin(); err != nil {
			t.Fatal("failed to begin transaction", err)
		}

		_, err = s.Raw("INSERT INTO User(Name) VALUES (?)", "Jack").Exec()
		if err != nil {
			t.Fatal("failed to insert", err)
		}

		if err := s.Rollback(); err != nil {
			t.Fatal("failed to rollback", err)
		}

		newCount := countUsers(t, s)
		if newCount != count {
			t.Fatalf("expected %d users after rollback, got %d", count, newCount)
		}
	})

	// Test Transaction helper
	t.Run("TestTransactionHelper", func(t *testing.T) {
		countBefore := countUsers(t, s)

		// Successful transaction
		err := s.Transaction(func(s *Session) error {
			_, err := s.Raw("INSERT INTO User(Name) VALUES (?)", "Alice").Exec()
			return err
		})

		if err != nil {
			t.Fatal("transaction failed", err)
		}

		countAfter := countUsers(t, s)
		if countAfter != countBefore+1 {
			t.Fatalf("expected %d users, got %d", countBefore+1, countAfter)
		}

		// Failed transaction
		countBefore = countUsers(t, s)
		err = s.Transaction(func(s *Session) error {
			// First insert works
			_, err := s.Raw("INSERT INTO User(Name) VALUES (?)", "Bob").Exec()
			if err != nil {
				return err
			}

			// Return an error to trigger rollback
			return sql.ErrConnDone
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		countAfter = countUsers(t, s)
		if countAfter != countBefore {
			t.Fatalf("expected %d users after rollback, got %d", countBefore, countAfter)
		}
	})
}

func countUsers(t *testing.T, s *Session) int {
	var count int
	row := s.Raw("SELECT COUNT(*) FROM User").QueryRow()
	if err := row.Scan(&count); err != nil {
		t.Fatal("failed to count users", err)
	}
	return count
}
