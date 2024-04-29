package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	MUSICDASH_SPOTIFY_CLIENT_ID = os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID")
	MUSICDASH_SPOTIFY_SECRET    = os.Getenv("MUSICDASH_SPOTIFY_SECRET")
	MUSICDASH_DATABASE_URL      = os.Getenv("MUSICDASH_DATABASE_URL")
)

// One instance of a Db{} database object is present per running program.
// The pointer to that main one is declared here in the "db" package. It is unexported, and is
// initially nil.
var dbInstance *Db

// An object representing a database connection.
type Db struct {
	pool *pgxpool.Pool
}

// Return a valid connected instance of the Db database object. This simply returns the global
// ptr to an existing instance, and instantiates it if already isn't.
func Acquire() *Db {
	if dbInstance == nil {
		pool, err := pgxpool.New(context.TODO(), MUSICDASH_DATABASE_URL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error acquiring database pgxpool connection: ", err)
			os.Exit(1)
		}

		dbInstance = &Db{pool}
	}

	return dbInstance
}

func (db *Db) Close() {
	db.pool.Close()
}

// return the underlying pool
func (db *Db) Pool() *pgxpool.Pool {
	return db.pool
}
