package db

import "time"

const (
	RegularAlbum AlbumType = iota
	Compilation
	Single
)

type AlbumType int

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

type Artist struct {
	Name                 string
	Discography          []Album
	SpotifyId            string
	SpotifyURI           string
	SpotifyFollowerCount int
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
