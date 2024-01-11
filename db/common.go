package db

import (
	music "bool3max/musicdash/music"
	"context"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool() (*pgxpool.Pool, error) {
	return pgxpool.New(context.TODO(), os.Getenv("MUSICDASH_DATABASE_URL"))
}

// A ResourceProvider that pulls data in from the local database
type Db struct {
	pool *pgxpool.Pool
}

func (db *Db) Close() {
	db.pool.Close()
}

func (db *Db) Pool() *pgxpool.Pool {
	return db.pool
}

func IncludeGroupToString(group []music.AlbumType) string {
	as_strings := make([]string, 0, len(group))
	for _, g := range group {
		as_strings = append(as_strings, string(g))
	}

	return strings.Join(as_strings, ",")
}

func New() (*Db, error) {
	pool, err := NewPool()
	if err != nil {
		return nil, err
	}

	return &Db{pool: pool}, nil
}
