// the spotify package deals with interaction with the Spotify api
package spotify

import (
	"bool3max/musicdash/db"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	endpointTrack  = "https://api.spotify.com/v1/tracks/"
	endpointArtist = "https://api.spotify.com/v1/artists/"
	endpointAlbum  = "https://api.spotify.com/v1/albums/"
	endpointSearch = "https://api.spotify.com/v1/search/"
)

type apiAccessToken struct {
	Access_token string `json:"access_token"`
	Token_type   string `json:"token_type"`
	Expires_in   int    `json:"expires_in"`
}

type spotifyAuth struct {
	client_id, client_secret string
	token                    string
	validUntil               time.Time
	authHeader               string
}

func (auth *spotifyAuth) refresh() error {
	requestFormData := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {auth.client_id},
		"client_secret": {auth.client_secret},
	}.Encode()

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(requestFormData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New(response.Status)
	}

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var info apiAccessToken
	err = json.Unmarshal(responseBytes, &info)
	if err != nil {
		return err
	}

	auth.token = info.Access_token
	auth.validUntil = time.Now().Add(time.Second * time.Duration(info.Expires_in)).Add(-1 * time.Minute)
	auth.authHeader = "Bearer " + auth.token

	return nil
}

type api struct {
	// authentication information of the current client
	auth spotifyAuth
}

// returns whether a refresh was attempted, and an error if the refresh failed or not
// if a refresh hasnt been attempted error is always nil
func (spot *api) validateToken() (bool, error) {
	if time.Now().After(spot.auth.validUntil) {
		// current token has expired, refresh
		if err := spot.auth.refresh(); err != nil {
			return true, err
		}

		return true, nil
	}

	return false, nil
}

// Create and return a new pre-authenticated SpotifyAPI instance.
func NewAPI(client_id, client_secret string) (*api, error) {
	spot := &api{}

	spot.auth.client_id = client_id
	spot.auth.client_secret = client_secret

	err := spot.auth.refresh()
	if err != nil {
		return nil, err
	}

	return spot, nil
}

func (spot *api) jsonGetHelper(uri string, decodeTo any) error {
	// validate current access token
	if _, err := spot.validateToken(); err != nil {
		// current token expired but revalidation failed
		return err
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", spot.auth.authHeader)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New(response.Status)
	}

	if err := json.NewDecoder(response.Body).Decode(decodeTo); err != nil {
		return err
	}

	return nil
}

// helper function for retrieving a resource from the Spotify API
// based on its spotify ID. resourceType should be one of:
// "track", "album", "artist"
func (spot *api) getResource(resourceType, spotifyId string) (any, error) {

	// determine which endpoint url to use based on type of desired resource
	var endpoint string
	switch resourceType {
	case "track":
		endpoint = endpointTrack
	case "album":
		endpoint = endpointAlbum
	case "artist":
		endpoint = endpointArtist
	}

	var resourcePtr any
	switch resourceType {
	case "track":
		resourcePtr = &track{}
	case "artist":
		resourcePtr = &artist{}
	case "album":
		resourcePtr = &album{}
	}

	if err := spot.jsonGetHelper(endpoint+spotifyId, resourcePtr); err != nil {
		return nil, err
	}

	switch t := resourcePtr.(type) {
	case *track:
		return t.toDB(), nil
	case *artist:
		return t.toDB(), nil
	case *album:
		return t.toDB(), nil
	default:
		return nil, errors.New("invalid resource type")
	}
}

func (spot *api) Search(query string, limit int) (Search, error) {
	searchQuery := url.Values{
		"q":     {url.QueryEscape(query)},
		"limit": {strconv.Itoa(limit)},
		"type":  {"album,track,artist"},
	}.Encode()

	var searchResults searchResponse
	err := spot.jsonGetHelper(endpointSearch+"?"+searchQuery, &searchResults)

	if err != nil {
		return Search{}, err
	}

	search := Search{}
	for _, track := range searchResults.Tracks.Items {
		search.Tracks = append(search.Tracks, track.toDB())
	}

	for _, artist := range searchResults.Artists.Items {
		search.Artists = append(search.Artists, artist.toDB())
	}

	for _, album := range searchResults.Albums.Items {
		search.Albums = append(search.Albums, album.toDB())
	}

	return search, nil
}

func (spot *api) GetTrackById(id string) (*db.Track, error) {
	resource, err := spot.getResource("track", id)
	if err != nil {
		return nil, err
	}

	track := resource.(db.Track)
	return &track, nil
}

func (spot *api) GetSeveralTracksById(ids []string) ([]db.Track, error) {
	requestUrl := endpointTrack + "?" + url.Values{"ids": {strings.Join(ids, ",")}}.Encode()
	var result struct {
		Tracks []track `json:"tracks"`
	}
	err := spot.jsonGetHelper(requestUrl, &result)
	if err != nil {
		return nil, err
	}

	dbTracks := make([]db.Track, len(ids))
	for trackIdx, track := range result.Tracks {
		dbTracks[trackIdx] = track.toDB()
	}

	return dbTracks, nil
}

func (spot *api) GetTrackByMatch(iden string) (*db.Track, error) {
	// perform a search to obtain id of the desired track
	search, err := spot.Search(iden, 1)
	if err != nil {
		return nil, err
	}

	firstResultId := search.Tracks[0].SpotifyId

	resource, err := spot.getResource("track", firstResultId)
	if err != nil {
		return nil, err
	}

	track := resource.(db.Track)
	return &track, nil
}

func (spot *api) GetArtistById(id string, discogFillLevel int) (*db.Artist, error) {
	resource, err := spot.getResource("artist", id)
	if err != nil {
		return nil, err
	}

	artist := resource.(db.Artist)

	if discogFillLevel > 0 {
		if err := artist.FillDiscography(spot, discogFillLevel > 1); err != nil {
			return nil, err
		}
	}

	return &artist, nil
}

func (spot *api) GetArtistByMatch(iden string, discogFillLevel int) (*db.Artist, error) {
	search, err := spot.Search(iden, 1)
	if err != nil {
		return nil, err
	}

	firstResultId := search.Artists[0].SpotifyId
	resource, err := spot.getResource("artist", firstResultId)
	if err != nil {
		return nil, err
	}

	artist := resource.(db.Artist)
	if discogFillLevel > 0 {
		if err := artist.FillDiscography(spot, discogFillLevel > 1); err != nil {
			return nil, err
		}
	}
	return &artist, nil
}

func (spot *api) GetAlbumById(id string) (*db.Album, error) {
	resource, err := spot.getResource("album", id)
	if err != nil {
		return nil, err
	}

	album := resource.(db.Album)
	return &album, nil
}

func (spot *api) GetAlbumByMatch(iden string) (*db.Album, error) {
	search, err := spot.Search(iden, 1)
	if err != nil {
		return nil, err
	}

	firstResultId := search.Albums[0].SpotifyId

	resource, err := spot.getResource("album", firstResultId)
	if err != nil {
		return nil, err
	}

	album := resource.(db.Album)
	return &album, nil
}

// Given an Artist, return their complete discography. This is necessary given
// that api.GetArtistBy<IdentifierType> does not return any discography information.
// includeGroups specifies types of albums to include in the response and can contain
// any of the following, at most once: album, single, appears_on, compilation
// If includeGroups is nil, {"album"} is assumed
func (spot *api) GetArtistDiscography(artist *db.Artist, includeGroups []db.AlbumType) ([]db.Album, error) {
	if includeGroups == nil {
		includeGroups = []db.AlbumType{db.AlbumRegular}
	}

	var response struct {
		Items []album

		Href   string `json:"href"`
		Next   string `json:"next"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}

	queryParams := url.Values{
		"limit":          {"50"},
		"include_groups": {db.IncludeGroupToString(includeGroups)},
	}.Encode()

	err := spot.jsonGetHelper(endpointArtist+artist.SpotifyId+"/albums?"+queryParams, &response)
	if err != nil {
		return nil, err
	}

	converted := make([]db.Album, len(response.Items))
	for idx, album := range response.Items {
		converted[idx] = album.toDB()
	}

	return converted, nil
}

// Given an Album, return all of its tracks. This method is necessary
// considering that sometimes db.Album objects won't have their .Tracks[]
// filled, for example if they came from db.Artist.FillDiscography(), etc...
// The Spotify Web API's /albums/<id>/tracks *does* return info about each track
// on the album, but in the form of a SimplifiedTrackObject. For that reason,
// here we perform two requests: one to collect Spotify IDs of all the tracks on the album
// and another to collect detailed info about each of the tracks using the dedicated
// /tracks endpoint.
func (spot *api) GetAlbumTracklist(album *db.Album) ([]db.Track, error) {
	var response struct {
		Items []track

		Href   string `json:"href"`
		Next   string `json:"next"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}

	queryParams := url.Values{
		"limit": {"50"},
	}.Encode()

	err := spot.jsonGetHelper(endpointAlbum+album.SpotifyId+"/tracks?"+queryParams, &response)
	if err != nil {
		return nil, err
	}

	tracklistIds := make([]string, len(response.Items))
	for idx, track := range response.Items {
		tracklistIds[idx] = track.Id
	}

	albumTracks, err := spot.GetSeveralTracksById(tracklistIds)
	if err != nil {
		return []db.Track{}, err
	}

	return albumTracks, nil
}
