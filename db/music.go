package db

import (
	music "bool3max/musicdash/music"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrResourceNotPreserved = errors.New("resource not found in database")
)

// return a new *pgxpool.Pool connected to the local database

func (db *Db) GetTrackById(trackId string) (*music.Track, error) {
	track := new(music.Track)
	track.SpotifyId = trackId

	// id of belonging album
	var albumId string

	// track duration in milliseconds from database
	var trackDuration uint

	// query the base info about the track from the database
	row := db.pool.QueryRow(
		context.TODO(),
		`
			select title, duration, tracklistnum, discnum, popularity, spotifyuri, explicit, isrc, spotifyidalbum
			from spotify.track		
			where spotifyid=$1
		`,
		trackId,
	)

	err := row.Scan(
		&track.Title,
		&trackDuration,
		&track.TracklistNum,
		&track.DiscNum,
		&track.SpotifyPopularity,
		&track.SpotifyURI,
		&track.IsExplicit,
		&track.Isrc,
		&albumId,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrResourceNotPreserved
		}
		return nil, err
	}

	// convert millisecond uint duration to time.Duration
	track.Duration = time.Duration(trackDuration * 1e6)

	belongingAlbum, err := db.GetAlbumById(albumId)
	if err != nil {
		return nil, err
	}

	track.Album = *belongingAlbum

	// spotify ids of all artists on the track, main artist first
	sqlQueryTrackArtists := `
		select spotifyidartist
		from spotify.track
			inner join spotify.track_artist on (spotify.track.spotifyid=spotify.track_artist.spotifyidtrack)
		where spotifyid=$1
		order by spotify.track_artist.ismain desc
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
		artist, err := db.GetArtistById(artistId, 0, nil)
		if err != nil {
			return nil, err
		}

		track.Artists = append(track.Artists, *artist)
	}

	return track, nil
}

func (db *Db) GetSeveralTracksById(ids []string) ([]music.Track, error) {
	tracks := make([]music.Track, len(ids))
	for idx, trackId := range ids {
		track, err := db.GetTrackById(trackId)
		if err != nil {
			return []music.Track{}, err
		}
		tracks[idx] = *track
	}

	return tracks, nil
}

func (db *Db) GetAlbumById(albumId string) (*music.Album, error) {
	album := new(music.Album)
	album.SpotifyId = albumId

	sqlQueryBaseInfo := `
		select title, counttracks, releasedate, spotifyuri, type, upc
		from spotify.album	
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
		&album.Upc,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrResourceNotPreserved
		}

		return nil, err
	}

	switch albumType {
	case "album":
		album.Type = music.AlbumRegular
	case "compilation":
		album.Type = music.AlbumCompilation
	case "single":
		album.Type = music.AlbumSingle
	}

	// spotify ids of all album artists, main artist first
	sqlQueryAlbumArtists := `
		select spotifyidartist 
		from spotify.album_artist
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
		artist, err := db.GetArtistById(artistId, 0, nil)
		if err != nil {
			return nil, err
		}

		album.Artists = append(album.Artists, *artist)
	}

	return album, nil
}

func (db *Db) GetSeveralAlbumsById(ids []string) ([]music.Album, error) {
	albums := make([]music.Album, len(ids))
	for idx, albumId := range ids {
		track, err := db.GetAlbumById(albumId)
		if err != nil {
			return []music.Album{}, err
		}
		albums[idx] = *track
	}

	return albums, nil
}

func (db *Db) GetArtistById(artistId string, discogFillLevel int, albumTypes []music.AlbumType) (*music.Artist, error) {
	artist := new(music.Artist)
	artist.SpotifyId = artistId
	sqlQueryBaseInfo := `
		select name, spotifyuri, followers
		from spotify.artist
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
		artist.FillDiscography(db, albumTypes, discogFillLevel > 1)
	}

	return artist, nil
}

func (db *Db) GetArtistDiscography(artist *music.Artist, includeGroups []music.AlbumType) ([]music.Album, error) {
	if includeGroups == nil {
		includeGroups = []music.AlbumType{music.AlbumRegular}
	}

	discog := make([]music.Album, 0)

	sqlQueryDiscog := `
		select spotifyidalbum
		from spotify.album_artist
			inner join spotify.album on (spotify.album_artist.spotifyidalbum=spotify.album.spotifyid)	
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

func (db *Db) GetAlbumTracklist(album *music.Album) ([]music.Track, error) {
	tracklist := make([]music.Track, album.CountTracks)

	// IDs of all tracks on the album
	sqlQueryTracks := `
		select spotifyid
		from spotify.track
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
func (db *Db) GetAlbumByMatch(iden string) (*music.Album, error) {
	sqlQuerySearch := `
		select spotifyid
		from spotify.album
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

func (db *Db) GetArtistByMatch(iden string, discogFillLevel int, albumTypes []music.AlbumType) (*music.Artist, error) {
	sqlQuerySearch := `
		select spotifyid
		from spotify.artist
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

	return db.GetArtistById(spotifyId, discogFillLevel, albumTypes)
}

func (db *Db) GetTrackByMatch(iden string) (*music.Track, error) {
	sqlQuerySearch := `
		select spotifyid
		from spotify.track
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
