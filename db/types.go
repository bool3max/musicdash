package db

import (
	"context"
	"fmt"
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

// Obtain an artist's complete discography using the specified provider
// after which it will be available in artist.Discography.
func (artist *Artist) FillDiscography(provider ResourceProvider) error {
	discog, err := provider.GetArtistDiscography(artist, nil)
	if err != nil {
		return err
	}

	artist.Discography = discog
	return nil
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

func (artist *Artist) Preserve(ctx context.Context, pool *pgxpool.Pool, preserveDiscography bool) error {
	sqlQuery := `
		insert into spotify_artist
		(spotifyid, name, spotifyuri, followers) 
		values ($1, $2, $3, $4)
	`

	// insert base info of artist
	_, err := pool.Exec(
		ctx,
		sqlQuery,
		artist.SpotifyId,
		artist.Name,
		artist.SpotifyURI,
		artist.SpotifyFollowerCount,
	)

	if err != nil {
		return err
	}

	// if desired, insert albums in artist's discography
	if preserveDiscography && artist.Discography != nil {
		fmt.Println("inserting discog...")
	}

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
	GetArtistById(string, bool) (*Artist, error)
	GetArtistByMatch(ResourceIdentifier, bool) (*Artist, error)
	GetArtistDiscography(*Artist, []string) ([]Album, error)
}
