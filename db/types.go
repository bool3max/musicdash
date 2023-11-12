package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlbumType int

const (
	RegularAlbum AlbumType = iota
	Compilation
	Single
)

type Resource interface {
	// check if the corresponding resource is preserved
	// in the local database
	IsPreserved() (bool, error)
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

func (track *Track) IsPreserved(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	row := pool.QueryRow(ctx, "select spotifyid from spotify_track where spotifyid=$1", track.SpotifyId)

	var id string
	err := row.Scan(&id)

	switch err {
	case pgx.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}

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

	switch err {
	case pgx.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}

func (artist *Artist) Preserve(ctx context.Context, pool *pgxpool.Pool) error {
	return nil
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

func (album *Album) IsPreserved(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	row := pool.QueryRow(ctx, "select spotifyid from spotify_album where spotifyid=$1", album.SpotifyId)

	var id string
	err := row.Scan(&id)

	switch err {
	case pgx.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
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
