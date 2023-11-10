package spotify

import (
	"bool3max/musicdash/db"
	"time"
)

// objects returned by the spotify API that are eventually converted to "real" application
// objects used throughout

// a resource returned by the Spotify API that can be converted
// into a "real" database resource
type track struct {
	Id         string
	Name       string
	Album      album
	Artists    []artist
	SpotifyURI string `json:"uri"`
	DurationMS int    `json:"duration_ms"`
	TrackNum   int    `json:"track_number"`
	Explicit   bool
	Popularity int
}

func (track track) toDB() db.Track {
	dbArtists := make([]db.Artist, len(track.Artists))
	for idx, spotifyArtist := range track.Artists {
		dbArtists[idx] = spotifyArtist.toDB()
	}

	// the duration over the api arrives as an integer representing
	// the duration of the track in milliseconds. in order to parse
	// this to a time.Duration type, i had to convert the duration
	// number to a string and append "ms" to the end to know
	// so ParseDuration knows the unit of time
	return db.Track{
		Title:             track.Name,
		Duration:          time.Duration(track.DurationMS * 1e6),
		TracklistNum:      track.TrackNum,
		Album:             track.Album.toDB(),
		Artists:           dbArtists,
		SpotifyId:         track.Id,
		SpotifyURI:        track.SpotifyURI,
		SpotifyPopularity: track.Popularity,
	}
}

type artist struct {
	Id        string
	Name      string
	Followers struct {
		Total int
	}
	Images     []image
	Populairy  int
	SpotifyURI string `json:"uri"`
}

func (artist artist) toDB() db.Artist {
	return db.Artist{
		Name:                 artist.Name,
		SpotifyId:            artist.Id,
		SpotifyURI:           artist.SpotifyURI,
		SpotifyFollowerCount: artist.Followers.Total,
	}
}

type album struct {
	Type        string `json:"album_type"`
	CountTracks int    `json:"total_tracks"`
	Id          string
	Name        string
	ReleaseDate string `json:"release_date"`
	Artists     []artist
	Tracks      struct {
		Items []track
	}
	Images     []image
	SpotifyURI string `json:"spotify_uri"`
}

func (album album) toDB() db.Album {
	dbArtists := make([]db.Artist, len(album.Artists))
	dbTracks := make([]db.Track, len(album.Tracks.Items))

	for idx, spotifyArtist := range album.Artists {
		dbArtists[idx] = spotifyArtist.toDB()
	}

	for idx, spotifyTrack := range album.Tracks.Items {
		dbTracks[idx] = spotifyTrack.toDB()
	}

	releaseDate, _ := time.Parse(time.DateOnly, album.ReleaseDate)
	var albumType db.AlbumType
	switch album.Type {
	case "album":
		albumType = db.RegularAlbum
	case "compilation":
		albumType = db.Compilation
	case "single":
		albumType = db.Single
	}

	return db.Album{
		Title:            album.Name,
		CountTracks:      album.CountTracks,
		Artists:          dbArtists,
		Tracks:           dbTracks,
		ReleaseDate:      releaseDate,
		SpotifyId:        album.Id,
		SpotifyURI:       album.SpotifyURI,
		SpotifyAlbumType: albumType,
	}
}

type image struct {
	Width, Height int
	Url           string
}

type search struct {
	Tracks struct {
		Items []track

		Href   string `json:"href"`
		Next   string `json:"next"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
		Total  int    `json:"total"`
	}

	Artists struct {
		Items []artist

		Href   string `json:"href"`
		Next   string `json:"next"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
		Total  int    `json:"total"`
	}

	Albums struct {
		Items []album

		Href   string `json:"href"`
		Next   string `json:"next"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
}
