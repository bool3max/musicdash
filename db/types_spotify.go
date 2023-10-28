package db

type Resource interface {
	preserve()
	isPreserved() (bool, error)
}

type Image struct {
	Width, Height int
	Url           string
}

type Track struct {
	// the fields below are returned by the /search API endpoint
	Id         string
	Name       string
	Artists    []Artist
	Album      Album
	SpotifyURI string `json:"uri"`
	DurationMS int    `json:"duration_ms"`
	TrackNum   int    `json:"track_number"`
	Popularity int
}

func (track *Track) preserve() {

}

func (track *Track) isPreserved() (bool, error) {
	return false, nil
}

type Artist struct {
	Id         string
	Name       string
	Images     []Image
	SpotifyURI string `json:"uri"`
}

func (artist *Artist) preserve() {

}

func (artist *Artist) isPreserved() (bool, error) {
	return false, nil
}

type Album struct {
	Type        string `json:"album_type"`
	CountTracks int    `json:"total_tracks"`
	Id          string
	Name        string
	ReleaseDate string
	Artists     []Artist
	Images      []Image
	SpotifyURI  string `json:"spotify_uri"`
}

func (album *Album) preserve() {

}

func (album *Album) isPreserved() (bool, error) {
	return false, nil
}
