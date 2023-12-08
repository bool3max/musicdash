package db

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlbumType string

const (
	AlbumRegular     AlbumType = "album"
	AlbumCompilation AlbumType = "compilation"
	AlbumSingle      AlbumType = "single"
)

type Resource interface {
	// Check if the corresponding resource is preserved in the local database
	IsPreserved(context.Context, *pgxpool.Pool) (bool, error)

	// Preserve the corresponding resource to the local database
	// The third argument denotes whether to recurse and also preserve
	// children of the resource. Argument may be ignored if the resource
	// has no children.
	Preserve(context.Context, *pgxpool.Pool, bool) error
}

// A visual representation of a Spotify resource identified by a Spotify ID.
// MusicImage.Data[] stores binary data of the image and may be empty (nil) if the
// image hasn't yet been downloaded. MusicImage.Download() downloads the image
// data and populates MusicImage.Data[] and MusicImage.MimeType. Preserving the image
// with MusicImage.Preserve() downloads the image beforehand, if it hasn't already
// been downloaded.
type MusicImage struct {
	Width, Height int
	MimeType      string
	SpotifyId     string
	Url           string
	Data          []byte
}

// Download binary image data from img.Url and store it in img.Data.
// Stores the MimeType of the binary data in img.MimeType.
func (img *MusicImage) Download() error {
	resp, err := http.Get(img.Url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	img.Data, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	img.MimeType = resp.Header.Get("Content-Type")
	return nil
}

func (img *MusicImage) IsPreserved(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	row := pool.QueryRow(
		ctx,
		`
			select url
			from spotify.images
			where url=$1 and spotifyid=$2 and width=$3 and height=$4
		`,
		img.Url, img.SpotifyId, img.Width, img.Height,
	)

	var url string
	err := row.Scan(&url)

	switch err {
	case pgx.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}

func (img *MusicImage) Preserve(ctx context.Context, pool *pgxpool.Pool, recurse bool) error {
	if img.Data == nil {
		if err := img.Download(); err != nil {
			return err
		}
	}

	sqlQueryImg := `
		insert into spotify.images
		(spotifyid, url, width, height, data, mimetype)	
		values
		($1, $2, $3, $4, $5, $6)
	`

	_, err := pool.Exec(
		ctx,
		sqlQueryImg,
		img.SpotifyId,
		img.Url,
		img.Width,
		img.Height,
		img.Data,
		img.MimeType,
	)

	return err
}

type Track struct {
	Title             string
	Duration          time.Duration
	TracklistNum      int
	IsExplicit        bool
	Album             Album
	Artists           []Artist
	SpotifyId         string
	Isrc              string
	Ean               string
	Upc               string
	SpotifyURI        string
	SpotifyPopularity int
}

// Preserve the track into the local database. Preserving a track performs
// the following database operations:
//  1. stores the base info of the track into public.spotify_track
//  2. stores the performing artists into public.spotify_track_artist
//     , properly marking the main performing artist, and preserving
//     any performing artists that aren't already in the database
//  3. stores the belonging albums into public.spotify_track_album,
//     preserving any belonging albums that aren't preserved
func (track *Track) Preserve(ctx context.Context, pool *pgxpool.Pool, recurse bool) error {
	sqlQueryBaseInfo := `
		insert into spotify.track
		(spotifyid, title, duration, tracklistnum, explicit, popularity, spotifyuri, isrc, ean, upc)	
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		on conflict on constraint track_pk do update
		set title = $2, duration = $3, tracklistnum = $4, explicit = $5, popularity = $6, spotifyuri = $7, isrc = $8, ean = $9, upc = $10
	`

	_, err := pool.Exec(
		ctx,
		sqlQueryBaseInfo,
		track.SpotifyId,
		track.Title,
		track.Duration.Milliseconds(),
		track.TracklistNum,
		track.IsExplicit,
		track.SpotifyPopularity,
		track.SpotifyURI,
		track.Isrc,
		track.Ean,
		track.Upc,
	)

	if err != nil {
		return err
	}

	// perserve the performing artists
	sqlQueryPerformingArtist := `
		insert into spotify.track_artist
		(spotifyidtrack, spotifyidartist, ismain)	
		values ($1, $2, $3)
		on conflict on constraint track_artist_pk do nothing
	`
	for performingArtistIdx, performingArtist := range track.Artists {
		// check if the current performing artist is currently preserved
		// in the local database
		isPreserved, err := performingArtist.IsPreserved(ctx, pool)
		if err != nil {
			return err
		}

		// if not, obtain full version of it and preserve it
		if !isPreserved {
			if err := performingArtist.Preserve(ctx, pool, recurse); err != nil {
				return err
			}
		}

		// the main performing artist of a track is always the
		// first one in the artists array
		isMain := performingArtistIdx == 0
		_, err = pool.Exec(
			ctx,
			sqlQueryPerformingArtist,
			track.SpotifyId,
			performingArtist.SpotifyId,
			isMain,
		)

		if err != nil {
			return err
		}
	}

	sqlQueryBelongingAlbum := `
		insert into spotify.track_album
		(spotifyidtrack, spotifyidalbum)
		values ($1, $2)
		on conflict on constraint track_album_pk do nothing
	`

	// if the track's owning album is set, preserve the relationship
	// in the database
	// sometimes the track's .Album won't be set (empty struct)
	// for example when the track comes from an Album.Tracks[], in that case
	// the album is known in advanace and does not need to be stored in each
	// individual track
	if track.Album.SpotifyId != "" {
		albumIsPreserved, err := track.Album.IsPreserved(ctx, pool)
		if err != nil {
			return err
		}

		if !albumIsPreserved {
			if err = track.Album.Preserve(ctx, pool, recurse); err != nil {
				return err
			}
		}

		_, err = pool.Exec(
			ctx,
			sqlQueryBelongingAlbum,
			track.SpotifyId,
			track.Album.SpotifyId,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (track *Track) IsPreserved(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	row := pool.QueryRow(ctx, "select spotifyid from spotify.track where spotifyid=$1", track.SpotifyId)

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
	Images               []MusicImage
	SpotifyId            string
	SpotifyURI           string
	SpotifyFollowerCount int
}

// Obtain an artist's complete discography using the specified provider
// after which it will be available in artist.Discography.
// If fillTracklists is true, each one of the album's tracklists is also provided.
func (artist *Artist) FillDiscography(provider ResourceProvider, albumTypes []AlbumType, fillTracklists bool) error {
	discog, err := provider.GetArtistDiscography(artist, albumTypes)
	if err != nil {
		return err
	}

	if fillTracklists {
		for discogAlbumIdx := range discog {
			if err := discog[discogAlbumIdx].FillTracklist(provider); err != nil {
				return err
			}
		}
	}

	artist.Discography = discog
	return nil
}

func (artist *Artist) IsPreserved(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	row := pool.QueryRow(ctx, "select spotifyid from spotify.artist where spotifyid=$1", artist.SpotifyId)

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

func (artist *Artist) Preserve(ctx context.Context, pool *pgxpool.Pool, recurse bool) error {
	sqlQuery := `
		insert into spotify.artist
		(spotifyid, name, spotifyuri, followers) 
		values ($1, $2, $3, $4)
		on conflict on constraint artist_pk do update
		set name = $2, spotifyuri = $3, followers = $4
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

	// preserve the Artist's entire discography if told to recurse
	if recurse {
		for _, album := range artist.Discography {
			albumPreserved, err := album.IsPreserved(ctx, pool)
			if err != nil {
				return err
			}

			if albumPreserved {
				continue
			}

			err = album.Preserve(ctx, pool, recurse)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type Album struct {
	Title       string
	CountTracks int
	Artists     []Artist
	Tracks      []Track
	Images      []MusicImage
	ReleaseDate time.Time
	Isrc        string
	Ean         string
	Upc         string
	SpotifyId   string
	SpotifyURI  string
	Type        AlbumType
}

func (album *Album) Preserve(ctx context.Context, pool *pgxpool.Pool, recurse bool) error {
	sqlQueryBaseInfo := `
		insert into spotify.album
		(spotifyid, title, counttracks, releasedate, type, spotifyuri, isrc, ean, upc)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		on conflict on constraint album_pk do update
		set title = $2, counttracks = $3, releasedate = $4, type = $5, spotifyuri = $6, isrc = $7, ean = $8, upc = $9
	`

	_, err := pool.Exec(
		ctx,
		sqlQueryBaseInfo,
		album.SpotifyId,
		album.Title,
		album.CountTracks,
		album.ReleaseDate,
		album.Type,
		album.SpotifyURI,
		album.Isrc,
		album.Ean,
		album.Upc,
	)

	if err != nil {
		return err
	}

	sqlQueryPerformingArtist := `
		insert into spotify.album_artist
		(spotifyidartist, spotifyidalbum, ismain)
		values ($1, $2, $3)
		on conflict on constraint album_artist_pk do nothing
	`

	// create a relation in public.spotify_album_artist
	// for each performing artist in album.Artists
	for performingArtistIdx, performingArtist := range album.Artists {
		artistIsPreserved, err := performingArtist.IsPreserved(ctx, pool)
		if err != nil {
			return err
		}

		if !artistIsPreserved {
			if err = performingArtist.Preserve(ctx, pool, recurse); err != nil {
				return err
			}
		}

		isMain := performingArtistIdx == 0
		_, err = pool.Exec(
			ctx,
			sqlQueryPerformingArtist,
			performingArtist.SpotifyId,
			album.SpotifyId,
			isMain,
		)

		if err != nil {
			return err
		}
	}

	// if album has list of tracks and told to recurse,
	// preserve all of the tracks as well
	if recurse {
		for _, albumTrack := range album.Tracks {
			trackPreserved, err := albumTrack.IsPreserved(ctx, pool)
			if err != nil {
				return err
			}

			if trackPreserved {
				continue
			}

			if err = albumTrack.Preserve(ctx, pool, recurse); err != nil {
				return err
			}

			// create a relation in public.spotify_track_album
			// for the newly inserted track

			sqlQueryTrackAlbumRelation := `
				insert into spotify.track_album
				(spotifyidtrack, spotifyidalbum)
				values ($1, $2)
				on conflict on constraint track_album_pk do nothing
			`

			_, err = pool.Exec(
				ctx,
				sqlQueryTrackAlbumRelation,
				albumTrack.SpotifyId,
				album.SpotifyId,
			)

			if err != nil {
				return err
			}

		}
	}

	return nil
}

func (album *Album) IsPreserved(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	row := pool.QueryRow(ctx, "select spotifyid from spotify.album where spotifyid=$1", album.SpotifyId)

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

func (album *Album) FillTracklist(provider ResourceProvider) error {
	tracklist, err := provider.GetAlbumTracklist(album)
	if err != nil {
		return err
	}

	album.Tracks = tracklist
	return nil
}

type ResourceProvider interface {
	GetTrackById(string) (*Track, error)
	GetSeveralTracksById([]string) ([]Track, error)
	GetTrackByMatch(string) (*Track, error)

	// Getting an album by whatever method never fills in its tracklist.
	// In order to obtain a tracklist, use either ResourceProvider.GetAlbumTracklist()
	// or db.Album.FillTracklist().
	GetAlbumById(string) (*Album, error)
	GetSeveralAlbumsById([]string) ([]Album, error)
	GetAlbumByMatch(string) (*Album, error)

	// The second "int" argument in the next two methods denotes
	// whether a discography should be provided when fetching an artist.
	// The value 0 means fetch no discography. The value 1 means fetch only
	// the albums, but not their tracklists. The value 2 or above means fetch
	// both the discographies and the albums' associated tracklists.
	// Note that ResourceProvider implementations may always provide *more*
	// data than has been requested, but never less. So you may receive
	// an artist with a fully filled out discography despite specifying
	// GetArtistByMatch(..., 0), or a discography with both albums
	// and their respective filled out tracklists despite specifying
	// GetArtistByMatch(..., 1)
	GetArtistById(string, int, []AlbumType) (*Artist, error)
	GetArtistByMatch(string, int, []AlbumType) (*Artist, error)

	GetArtistDiscography(*Artist, []AlbumType) ([]Album, error)
	GetAlbumTracklist(*Album) ([]Track, error)
}
