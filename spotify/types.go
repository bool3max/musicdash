package spotify

import (
	music "bool3max/musicdash/music"
	"time"
)

type external_ids struct {
	Isrc string `json:"isrc"`
	Ean  string `json:"ean"`
	Upc  string `json:"upc"`
}

type track struct {
	Id          string
	Name        string
	Album       album
	Artists     []artist
	SpotifyURI  string `json:"uri"`
	DurationMS  int    `json:"duration_ms"`
	TrackNum    int    `json:"track_number"`
	DiscNum     int    `json:"disc_number"`
	Explicit    bool   `json:"explicit"`
	Popularity  int
	ExternalIds external_ids `json:"external_ids"`
}

func (track track) toDB() music.Track {
	dbArtists := make([]music.Artist, len(track.Artists))
	for idx, spotifyArtist := range track.Artists {
		dbArtists[idx] = spotifyArtist.toDB()
	}

	return music.Track{
		Title:             track.Name,
		Duration:          time.Duration(track.DurationMS * 1e6),
		TracklistNum:      track.TrackNum,
		DiscNum:           track.DiscNum,
		IsExplicit:        track.Explicit,
		Album:             track.Album.toDB(),
		Artists:           dbArtists,
		SpotifyId:         track.Id,
		Ean:               track.ExternalIds.Ean,
		Isrc:              track.ExternalIds.Isrc,
		Upc:               track.ExternalIds.Upc,
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
	Popularity int
	SpotifyURI string `json:"uri"`
}

func (artist artist) toDB() music.Artist {
	dbImages := make([]music.Image, len(artist.Images))
	for idx, img := range artist.Images {
		dbImages[idx] = img.toDB(artist.Id)
	}

	return music.Artist{
		Name:                 artist.Name,
		Images:               dbImages,
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
	ExternalIds external_ids `json:"external_ids"`
	Images      []image
	SpotifyURI  string `json:"uri"`
}

func (album album) toDB() music.Album {
	dbArtists := make([]music.Artist, len(album.Artists))
	dbTracks := make([]music.Track, len(album.Tracks.Items))
	dbImages := make([]music.Image, len(album.Images))

	for idx, spotifyArtist := range album.Artists {
		dbArtists[idx] = spotifyArtist.toDB()
	}

	for idx, spotifyTrack := range album.Tracks.Items {
		dbTracks[idx] = spotifyTrack.toDB()
	}

	for idx, img := range album.Images {
		dbImages[idx] = img.toDB(album.Id)
	}

	releaseDate, _ := time.Parse(time.DateOnly, album.ReleaseDate)

	// parse album type from string to db.AlbumType
	var albumType music.AlbumType
	switch album.Type {
	case "album":
		albumType = music.AlbumRegular
	case "compilation":
		albumType = music.AlbumCompilation
	case "single":
		albumType = music.AlbumSingle
	}

	return music.Album{
		Title:       album.Name,
		CountTracks: album.CountTracks,
		Artists:     dbArtists,
		Tracks:      dbTracks,
		ReleaseDate: releaseDate,
		Images:      dbImages,
		SpotifyId:   album.Id,
		SpotifyURI:  album.SpotifyURI,
		Isrc:        album.ExternalIds.Isrc,
		Ean:         album.ExternalIds.Ean,
		Upc:         album.ExternalIds.Upc,
		Type:        albumType,
	}
}

type image struct {
	Width, Height int
	Url           string
}

func (image image) toDB(spotifyId string) music.Image {
	return music.Image{
		Width:     image.Width,
		Height:    image.Height,
		Url:       image.Url,
		SpotifyId: spotifyId,
	}
}

// albums, tracks, and artist returned via SearchResults() usually contain
// less information that proper counterparts returned via
// .Get<Resource>By<IdentityType>
// SearchResults() should be used when a non-specific resource is being looked for,
// or when an ID of a specific resoure is required in order to obtain its
// complete version
type SearchResults struct {
	Albums  []music.Album
	Tracks  []music.Track
	Artists []music.Artist
}

// API response struct for the Spotify /search endpoint
type searchResponse struct {
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

type UserProfile struct {
	SpotifyId     string
	DisplayName   string
	Email         string
	FollowerCount uint
	ProfileUri    string
	ProfileUrl    string
	ProfileImages []music.Image
	Country       string
}

// info and metadata about the currently playing resource by an user
type CurrentlyPlaying struct {
	// The API returns info about currently playing resource even if it's PAUSED. This field indicates
	// if the user is currently playing something LIVE, i.e. not paused.
	IsPlayingNow bool
	Progress     time.Duration
	Track        music.Track
}

type Play struct {
	At    time.Time
	Track music.Track
}
