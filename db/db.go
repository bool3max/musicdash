package db

import (
	"context"
	"os"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool() (*pgxpool.Pool, error) {
	dburl := os.Getenv("MUSICDASH_DATABASE_URL")
	dbpool, err := pgxpool.New(context.TODO(), dburl)

	if err != nil {
		return nil, err
	}

	return dbpool, nil
}
