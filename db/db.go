package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrResourceNotPreserved = errors.New("resource not found in database")
)

// return a new *pgxpool.Pool connected to the local database
func NewPool() (*pgxpool.Pool, error) {
	return pgxpool.New(context.TODO(), os.Getenv("MUSICDASH_DATABASE_URL"))
}

// a ResourceProvider that pulls data in from the local database
type db struct {
	pool *pgxpool.Pool
}

func IncludeGroupToString(group []AlbumType) string {
	as_strings := make([]string, 0, len(group))
	for _, g := range group {
		as_strings = append(as_strings, string(g))
	}

	return strings.Join(as_strings, ",")
}

func (db *db) GetTrackById(trackId string) (*Track, error) {
	track := new(Track)
	track.SpotifyId = trackId

	// query the base info about the track from the database
	row := db.pool.QueryRow(
		context.TODO(),
		`
			select title, duration, tracklistnum, popularity, spotifyuri, explicit
			from spotify_track		
			where spotifyid=$1
		`,
		trackId,
	)

	// track duration in milliseconds from database
	var trackDuration uint

	err := row.Scan(
		&track.Title,
		&trackDuration,
		&track.TracklistNum,
		&track.SpotifyPopularity,
		&track.SpotifyURI,
		&track.IsExplicit,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrResourceNotPreserved
		}
		return nil, err
	}

	// convert millisecond uint duration to time.Duration
	track.Duration = time.Duration(trackDuration * 1e6)

	// spotify ids of all artists on the track, main artist first
	sqlQueryTrackArtists := `
		select spotifyidartist
		from spotify_track
			inner join spotify_track_artist on (spotify_track.spotifyid=spotify_track_artist.spotifyidtrack)
		where spotifyid=$1
		order by spotify_track_artist.ismain desc
	`

	rows, err := db.pool.Query(
		context.TODO(),
		sqlQueryTrackArtists,
		trackId,
	)

	if err != nil {
		return nil, err
	}

	artistIds, err := pgx.CollectRows[string](rows, pgx.RowTo)
	if err != nil {
		return nil, err
	}

	for _, artistId := range artistIds {
		artist, err := db.GetArtistById(artistId, 0)
		if err != nil {
			return nil, err
		}

		track.Artists = append(track.Artists, *artist)
	}

	return track, nil
}

func (db *db) GetSeveralTracksById(ids []string) ([]Track, error) {
	tracks := make([]Track, len(ids))
	for idx, trackId := range ids {
		track, err := db.GetTrackById(trackId)
		if err != nil {
			return []Track{}, err
		}
		tracks[idx] = *track
	}

	return tracks, nil
}

func (db *db) GetAlbumById(albumId string) (*Album, error) {
	album := new(Album)
	album.SpotifyId = albumId

	sqlQueryBaseInfo := `
		select title, counttracks, releasedate, spotifyuri, type
		from spotify_album	
		where spotifyid=$1
	`

	row := db.pool.QueryRow(
		context.TODO(),
		sqlQueryBaseInfo,
		albumId,
	)

	var albumType string
	err := row.Scan(
		&album.Title,
		&album.CountTracks,
		&album.ReleaseDate,
		&album.SpotifyURI,
		&albumType,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrResourceNotPreserved
		}

		return nil, err
	}

	switch albumType {
	case "regular":
		album.Type = AlbumRegular
	case "compilation":
		album.Type = AlbumCompilation
	case "single":
		album.Type = AlbumSingle
	}

	// spotify ids of all album artists, main artist first
	sqlQueryAlbumArtists := `
		select spotifyidartist 
		from spotify_album_artist
		where spotifyidalbum=$1
		order by ismain desc
	`

	rows, err := db.pool.Query(
		context.TODO(),
		sqlQueryAlbumArtists,
		albumId,
	)

	if err != nil {
		return nil, err
	}

	artistIds, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	for _, artistId := range artistIds {
		artist, err := db.GetArtistById(artistId, 0)
		if err != nil {
			return nil, err
		}

		album.Artists = append(album.Artists, *artist)
	}

	// spotify ids of all tracks on the album
	sqlQueryAlbumTracks := `
		select spotifyidtrack
		from spotify_track_album
			inner join spotify_track on (spotify_track_album.spotifyidtrack=spotify_track.spotifyid)
		where spotifyidalbum=$1
		order by spotify_track.tracklistnum asc
	`

	rows, err = db.pool.Query(
		context.TODO(),
		sqlQueryAlbumTracks,
		albumId,
	)

	if err != nil {
		return nil, err
	}

	trackIds, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	for _, trackId := range trackIds {
		track, err := db.GetTrackById(trackId)
		if err != nil {
			return nil, err
		}

		album.Tracks = append(album.Tracks, *track)
	}

	return album, nil
}

func (db *db) GetSeveralAlbumsById(ids []string) ([]Album, error) {
	albums := make([]Album, len(ids))
	for idx, albumId := range ids {
		track, err := db.GetAlbumById(albumId)
		if err != nil {
			return []Album{}, err
		}
		albums[idx] = *track
	}

	return albums, nil
}

func (db *db) GetArtistById(artistId string, discogFillLevel int) (*Artist, error) {
	artist := new(Artist)
	artist.SpotifyId = artistId
	sqlQueryBaseInfo := `
		select name, spotifyuri, followers
		from spotify_artist
		where spotifyid=$1
	`

	row := db.pool.QueryRow(
		context.TODO(),
		sqlQueryBaseInfo,
		artistId,
	)

	err := row.Scan(
		&artist.Name,
		&artist.SpotifyURI,
		&artist.SpotifyFollowerCount,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrResourceNotPreserved
		}
		return nil, err
	}

	if discogFillLevel > 0 {
		artist.FillDiscography(db, discogFillLevel > 1)
	}

	return artist, nil
}

func (db *db) GetArtistDiscography(artist *Artist, includeGroups []AlbumType) ([]Album, error) {
	if includeGroups == nil {
		includeGroups = []AlbumType{AlbumRegular}
	}

	discog := make([]Album, 0)

	sqlQueryDiscog := `
		select spotifyidalbum
		from spotify_album_artist
			inner join spotify_album on (spotify_album_artist.spotifyidalbum=spotify_album.spotifyid)	
		where spotifyidartist=$1 and ismain is true and type=any($2)
	`

	rows, err := db.pool.Query(
		context.TODO(),
		sqlQueryDiscog,
		artist.SpotifyId,
		includeGroups,
	)

	if err != nil {
		return nil, err
	}

	albumIds, err := pgx.CollectRows[string](rows, pgx.RowTo)
	if err != nil {
		return nil, err
	}

	for _, albumId := range albumIds {
		album, err := db.GetAlbumById(albumId)
		if err != nil {
			return nil, err
		}

		discog = append(discog, *album)
	}

	return discog, nil
}

func (db *db) GetAlbumTracklist(album *Album) ([]Track, error) {
	tracklist := make([]Track, album.CountTracks)

	sqlQueryTracks := `
		select spotifyidtrack
		from spotify_track_album
		where spotifyidalbum=$1
		limit $2
	`

	rows, err := db.pool.Query(
		context.TODO(),
		sqlQueryTracks,
		album.SpotifyId,
		album.CountTracks,
	)

	if err != nil {
		return nil, err
	}

	trackIds, err := pgx.CollectRows[string](rows, pgx.RowTo)
	if err != nil {
		return nil, err
	}

	for trackIdx, trackId := range trackIds {
		track, err := db.GetTrackById(trackId)
		if err != nil {
			return nil, err
		}

		tracklist[trackIdx] = *track
	}

	return tracklist, nil
}

// Get<Resource>ByMatch methods of the "db" resource provider search exclusively
// based on the name of the resource being searched for, whereas the same methods
// of the Spotify provider search the entire spotify catalog (using its /search endpoint)
// and return the first result of the requested type
func (db *db) GetAlbumByMatch(iden string) (*Album, error) {
	sqlQuerySearch := `
		select spotifyid
		from spotify_album
		order by levenshtein($1, title)
		limit 1
	`

	row := db.pool.QueryRow(
		context.TODO(),
		sqlQuerySearch,
		iden,
	)

	var spotifyId string
	err := row.Scan(&spotifyId)

	if err != nil {
		return nil, err
	}

	return db.GetAlbumById(spotifyId)
}

func (db *db) GetArtistByMatch(iden string, discogFillLevel int) (*Artist, error) {
	sqlQuerySearch := `
		select spotifyid
		from spotify_artist
		order by levenshtein($1, name)
		limit 1
	`

	row := db.pool.QueryRow(
		context.TODO(),
		sqlQuerySearch,
		iden,
	)

	var spotifyId string
	err := row.Scan(&spotifyId)

	if err != nil {
		return nil, err
	}

	return db.GetArtistById(spotifyId, discogFillLevel)
}

func (db *db) GetTrackByMatch(iden string) (*Track, error) {
	sqlQuerySearch := `
		select spotifyid
		from spotify_track
		order by levenshtein($1, title)
		limit 1
	`

	row := db.pool.QueryRow(
		context.TODO(),
		sqlQuerySearch,
		iden,
	)

	var spotifyId string
	err := row.Scan(&spotifyId)

	if err != nil {
		return nil, err
	}

	return db.GetTrackById(spotifyId)
}

func New() (*db, error) {
	pool, err := NewPool()
	if err != nil {
		return nil, err
	}

	return &db{pool: pool}, nil
}
