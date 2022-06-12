package db

import (
	"database/sql"
)

type DB struct {
	db *sql.DB
}

func New() (*DB, error) {
	db, err := sql.Open("sqlite3_with_extensions", ":memory")
	if err != nil {
		return nil, err
	}
	s := &DB{
		db: db,
	}
	return s, nil

}

func (s *DB) Close() error {
	return s.db.Close()
}
