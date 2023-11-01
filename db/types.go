package db

import "time"

const (
	RegularAlbum AlbumType = iota
	Compilation
	Single
)

type AlbumType int

type Track struct {
	Title        string
	Duration     time.Duration
	TracklistNum int
	Album        Album
	Artists      []Artist
	Spotify      struct {
		Id         string
		URI        string
		Popularity int
	}
}

type Artist struct {
	Name        string
	Discography []Album
	Spotify     struct {
		Id            string
		URI           string
		FollowerCount int
	}
}

type Album struct {
	Title       string
	CountTracks int
	Artists     []Artist
	Tracks      []Track
	ReleaseDate time.Time
	Spotify     struct {
		Id        string
		URI       string
		AlbumType AlbumType
	}
}

// used to search for an artist, album, or track by name
type ResourceIdentifier struct {
	Title, Artist string
}

type ResourceProvider interface {
	GetTrackById(string) Track
	GetTrackByMatch(ResourceIdentifier) Track
	GetAlbumById(string) Album
	GetAlbumByMatch(ResourceIdentifier) Album
	GetArtistById(string) Artist
	GetArtistByMatch(ResourceIdentifier) Artist
}
