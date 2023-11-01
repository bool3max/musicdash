package spotify

// objects returned by the spotify API that are eventually converted to "real" application
// objects used throughout

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

type album struct {
	Type        string `json:"album_type"`
	CountTracks int    `json:"total_tracks"`
	Id          string
	Name        string
	ReleaseDate string
	Artists     []artist
	Tracks      struct {
		items []track
	}
	Images     []image
	SpotifyURI string `json:"spotify_uri"`
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
