package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RegularAlbum AlbumType = iota
	Compilation
	Single
)

type AlbumType int

type Resource interface {
	// check if the corresponding resource is preserved
	// in the local database
	isPreserved() (bool, error)
}

type Track struct {
	Title             string
	Duration          time.Duration
	TracklistNum      int
	Album             Album
	Artists           []Artist
	SpotifyId         string
	SpotifyURI        string
	SpotifyPopularity int
}

// func (track *Track) isPreserved(pool *pgxpool.Pool) (bool, error) {
// 	return false, nil
// }

type Artist struct {
	Name                 string
	Discography          []Album
	SpotifyId            string
	SpotifyURI           string
	SpotifyFollowerCount int
}

func (artist *Artist) IsPreserved(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	row := pool.QueryRow(ctx, "select spotifyid from spotify_artist where spotifyid=$1", artist.SpotifyId)
	var id string
	err := row.Scan(&id)
	if err == pgx.ErrNoRows {
		return false, nil
	} else if err == nil {
		return true, nil
	} else {
		return false, err
	}
}

type Album struct {
	Title            string
	CountTracks      int
	Artists          []Artist
	Tracks           []Track
	ReleaseDate      time.Time
	SpotifyId        string
	SpotifyURI       string
	SpotifyAlbumType AlbumType
}

// used to search for an artist, album, or track by name
type ResourceIdentifier struct {
	Title, Artist string
}

type ResourceProvider interface {
	GetTrackById(string) (*Track, error)
	GetTrackByMatch(ResourceIdentifier) (*Track, error)
	GetAlbumById(string) (*Album, error)
	GetAlbumByMatch(ResourceIdentifier) (*Album, error)
	GetArtistById(string) (*Artist, error)
	GetArtistByMatch(ResourceIdentifier) (*Artist, error)
}
