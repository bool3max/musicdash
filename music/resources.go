// The music package houses common types and methods for music-related resources
// (tracks, albums, artists, images, etc...)
package music

import (
	"context"
	"io"
	"net/http"
	"strings"
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

func IncludeGroupToString(group []AlbumType) string {
	as_strings := make([]string, len(group))
	for idx, g := range group {
		as_strings[idx] = string(g)
	}

	return strings.Join(as_strings, ",")
}

// A visual representation of a Spotify resource identified by a Spotify ID.
// Image.Data[] stores binary data of the image and may be empty (nil) if the
// image hasn't yet been downloaded. Image.Download() downloads the image
// data and populates Image.Data[] and Image.MimeType. Preserving the image
// with Image.Preserve() downloads the image beforehand, if it hasn't already
// been downloaded.
type Image struct {
	Width, Height int
	MimeType      string
	SpotifyId     string
	Url           string
	Data          []byte
}

// Download binary image data from img.Url and store it in img.Data.
// Stores the MimeType of the binary data in img.MimeType.
func (img *Image) Download() error {
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

func (img *Image) IsPreserved(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
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

func (img *Image) Preserve(ctx context.Context, pool *pgxpool.Pool, recurse bool) error {
	if img.Data == nil {
		if err := img.Download(); err != nil {
			return err
		}
	}

	sqlQueryImg := `
		insert into spotify.images
		(spotifyid, url, width, height, data, mimetype)	
		values
		(@spotifyId, @url, @width, @height, @data, @mimeType)
	`

	_, err := pool.Exec(
		ctx,
		sqlQueryImg,
		pgx.NamedArgs{
			"spotifyId": img.SpotifyId,
			"url":       img.Url,
			"width":     img.Width,
			"height":    img.Height,
			"data":      img.Data,
			"mimeType":  img.MimeType,
		},
	)

	return err
}

type Track struct {
	Title             string
	Duration          time.Duration
	TracklistNum      int
	DiscNum           int
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
func (track *Track) Preserve(ctx context.Context, pool *pgxpool.Pool, recurse bool) error {
	// preserve the track's belonging album if it isn't already so
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
	}

	sqlQueryBaseInfo := `
		insert into spotify.track
		(spotifyid, title, duration, tracklistnum, discnum, explicit, popularity, spotifyuri, isrc, ean, upc, spotifyidalbum)	
		values (@spotifyId, @title, @duration, @tracklistNum, @discNum, @explicit, @popularity, @spotifyUri, @isrc, @ean, @upc, @spotifyIdAlbum)
		on conflict on constraint track_pk do update
		set title = @title, duration = @duration, tracklistnum = @tracklistNum, discnum = @discNum, explicit = @explicit, popularity = @popularity, spotifyuri = @spotifyUri, isrc = @isrc, ean = @ean, upc = @upc, spotifyidalbum=@spotifyIdAlbum
	`

	_, err := pool.Exec(
		ctx,
		sqlQueryBaseInfo,
		pgx.NamedArgs{
			"spotifyId":      track.SpotifyId,
			"title":          track.Title,
			"duration":       track.Duration.Milliseconds(),
			"tracklistNum":   track.TracklistNum,
			"discNum":        track.DiscNum,
			"explicit":       track.IsExplicit,
			"popularity":     track.SpotifyPopularity,
			"spotifyUri":     track.SpotifyURI,
			"isrc":           track.Isrc,
			"ean":            track.Ean,
			"upc":            track.Upc,
			"spotifyIdAlbum": track.Album.SpotifyId,
		},
	)

	if err != nil {
		return err
	}

	// perserve the performing artists
	sqlQueryPerformingArtist := `
		insert into spotify.track_artist
		(spotifyidtrack, spotifyidartist, ismain)	
		values (@spotifyIdTrack, @spotifyIdArtist, @isMain)
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
			pgx.NamedArgs{
				"spotifyIdTrack":  track.SpotifyId,
				"spotifyIdArtist": performingArtist.SpotifyId,
				"isMain":          isMain,
			},
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
	Images               []Image
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
		values (@spotifyId, @name, @spotifyUri, @followers)
		on conflict on constraint artist_pk do update
		set name = $2, spotifyuri = $3, followers = $4
	`

	// insert base info of artist
	_, err := pool.Exec(
		ctx,
		sqlQuery,
		pgx.NamedArgs{
			"spotifyId":  artist.SpotifyId,
			"name":       artist.Name,
			"spotifyUri": artist.SpotifyURI,
			"followers":  artist.SpotifyFollowerCount,
		},
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
	Images      []Image
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
		values (@spotifyId, @title, @countTracks, @releaseDate, @type, @spotifyUri, @isrc, @ean, @upc)
		on conflict on constraint album_pk do update
		set title = @title, counttracks = @countTracks, releasedate = @releaseDate, type = @type, spotifyuri = @spotifyUri, isrc = @isrc, ean = @ean, upc = @upc 
	`

	_, err := pool.Exec(
		ctx,
		sqlQueryBaseInfo,
		pgx.NamedArgs{
			"spotifyId":   album.SpotifyId,
			"title":       album.Title,
			"countTracks": album.CountTracks,
			"releaseDate": album.ReleaseDate,
			"type":        album.Type,
			"spotifyUri":  album.SpotifyURI,
			"isrc":        album.Isrc,
			"ean":         album.Ean,
			"upc":         album.Upc,
		},
	)

	if err != nil {
		return err
	}

	sqlQueryPerformingArtist := `
		insert into spotify.album_artist
		(spotifyidartist, spotifyidalbum, ismain)
		values (@spotifyIdArtist, @spotifyIdAlbum, @isMain)
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
			pgx.NamedArgs{
				"spotifyIdArtist": performingArtist.SpotifyId,
				"spotifyIdAlbum":  album.SpotifyId,
				"isMain":          isMain,
			},
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
