package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	// import sqlite.
	_ "github.com/mattn/go-sqlite3"
)

const (
	createCharactersTableSQL = `CREATE TABLE IF NOT EXISTS characters (
		id number PRIMARY KEY,
		name TEXT NOT NULL,
		realm TEXT NOT NULL,
		class TEXT NOT NULL,
		score TEXT NOT NULL,
		tank_score TEXT NOT NULL,
		heal_score TEXT NOT NULL,
		dps_score TEXT NOT NULL,
		date_updated INTEGER DEFAULT (unixepoch()),
		date_created INTEGER DEFAULT (unixepoch())
	);`

	createCharactersTableTriggersSQL = `CREATE TRIGGER IF NOT EXISTS update_characters_date_updated
		AFTER UPDATE ON characters
		FOR EACH ROW
		BEGIN
			UPDATE characters SET date_updated = unixepoch() WHERE id = OLD.id;
		END;`
)

var (
	ErrOpeningFile = errors.New("error opening file")
	ErrNoDatabase  = errors.New("db is nil")
)

// Database defines the interface for database operations
type Database interface {
	Query(ctx context.Context, query string, args ...any) error
	QueryRows(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	Close() error
}

// CharacterRepository defines the interface for character operations
type CharacterRepository interface {
	Insert(ctx context.Context, character *Character) error
	Update(ctx context.Context, character *Character) error
	Delete(ctx context.Context, character *Character) error
	GetCharacter(ctx context.Context, name, realm string) (Character, error)
	CheckCharacterExists(ctx context.Context, name, realm string) (bool, error)
	ListCharacters(ctx context.Context, limit int) ([]Character, error)
}

// SQLiteDB implements the Database interface
type SQLiteDB struct {
	db *sql.DB
}

// NewSQLiteDB creates a new SQLite database instance
func NewSQLiteDB(dbLocation string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite3", dbLocation)
	if err != nil {
		return nil, errors.Join(ErrOpeningFile, err)
	}
	return &SQLiteDB{db: db}, nil
}

// Init initializes the database with required tables
func (s *SQLiteDB) Init(ctx context.Context) error {
	slog.DebugContext(ctx, "connecting to database")

	if err := s.initTable(ctx); err != nil {
		return err
	}

	slog.DebugContext(ctx, "database ready")
	return nil
}

func (s *SQLiteDB) initTable(ctx context.Context) error {
	if s.db == nil {
		return ErrNoDatabase
	}

	if err := s.Query(ctx, createCharactersTableSQL); err != nil {
		return err
	}

	return s.Query(ctx, createCharactersTableTriggersSQL)
}

func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteDB) Query(ctx context.Context, query string, args ...any) error {
	if s.db == nil {
		return ErrNoDatabase
	}

	slog.DebugContext(ctx, "executing query", "query", query, "args", args)
	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, args...)
	return err
}

func (s *SQLiteDB) QueryRows(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if s.db == nil {
		return nil, ErrNoDatabase
	}

	slog.DebugContext(ctx, "executing query", "query", query, "args", args)
	return s.db.QueryContext(ctx, query, args...)
}
