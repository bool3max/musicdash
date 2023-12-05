// The spotify package defines the Spotify ResourceProvider which obtains
// data from the Spotify Web API, as well as some constants.
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

type apiAuth struct {
	client_id, client_secret string
	token                    string
	validUntil               time.Time
	authHeader               string
}

func (auth *apiAuth) refresh() error {
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

type spotify struct {
	// authentication information of the current client
	auth apiAuth
}

// returns whether a refresh was attempted, and an error if the refresh failed or not
// if a refresh hasnt been attempted error is always nil
func (spot *spotify) validateToken() (bool, error) {
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
func NewAPI(client_id, client_secret string) (*spotify, error) {
	spot := &spotify{}

	spot.auth.client_id = client_id
	spot.auth.client_secret = client_secret

	err := spot.auth.refresh()
	if err != nil {
		return nil, err
	}

	return spot, nil
}

func (spot *spotify) jsonGetHelper(uri string, decodeTo any) error {
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

func (spot *spotify) Search(query string, limit int) (Search, error) {
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

	var search Search
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

func (spot *spotify) GetTrackById(id string) (*db.Track, error) {
	var track track
	if err := spot.jsonGetHelper(endpointTrack+id, &track); err != nil {
		return nil, err
	}

	dbTrack := track.toDB()

	return &dbTrack, nil
}

func (spot *spotify) GetSeveralTracksById(ids []string) ([]db.Track, error) {
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

func (spot *spotify) GetTrackByMatch(iden string) (*db.Track, error) {
	// perform a search to obtain id of the desired track
	search, err := spot.Search(iden, 1)
	if err != nil {
		return nil, err
	}

	firstResultId := search.Tracks[0].SpotifyId

	return spot.GetTrackById(firstResultId)
}

func (spot *spotify) GetArtistById(id string, discogFillLevel int) (*db.Artist, error) {
	var artist artist
	if err := spot.jsonGetHelper(endpointArtist+id, &artist); err != nil {
		return nil, err
	}

	dbArtist := artist.toDB()

	if discogFillLevel > 0 {
		if err := dbArtist.FillDiscography(spot, discogFillLevel > 1); err != nil {
			return nil, err
		}
	}

	return &dbArtist, nil
}

func (spot *spotify) GetArtistByMatch(iden string, discogFillLevel int) (*db.Artist, error) {
	search, err := spot.Search(iden, 1)
	if err != nil {
		return nil, err
	}

	firstResultId := search.Artists[0].SpotifyId
	artist, err := spot.GetArtistById(firstResultId, discogFillLevel)
	if err != nil {
		return nil, err
	}

	return artist, nil
}

func (spot *spotify) GetAlbumById(id string) (*db.Album, error) {
	var album album
	if err := spot.jsonGetHelper(endpointAlbum+id, &album); err != nil {
		return nil, err
	}

	dbAlbum := album.toDB()

	return &dbAlbum, nil
}

func (spot *spotify) GetSeveralAlbumsById(ids []string) ([]db.Album, error) {
	requestUrl := endpointAlbum + "?" + url.Values{"ids": {strings.Join(ids, ",")}}.Encode()
	var result struct {
		Albums []album `json:"albums"`
	}
	err := spot.jsonGetHelper(requestUrl, &result)
	if err != nil {
		return nil, err
	}

	dbAlbums := make([]db.Album, len(ids))
	for albumIdx, album := range result.Albums {
		dbAlbums[albumIdx] = album.toDB()
	}

	return dbAlbums, nil
}

func (spot *spotify) GetAlbumByMatch(iden string) (*db.Album, error) {
	search, err := spot.Search(iden, 1)
	if err != nil {
		return nil, err
	}

	firstResultId := search.Albums[0].SpotifyId

	return spot.GetAlbumById(firstResultId)
}

// Given an Artist, return their complete discography. This is necessary given
// that api.GetArtistBy<IdentifierType> does not return any discography information.
// includeGroups specifies types of albums to include in the response and can contain
// any of the following, at most once: album, single, appears_on, compilation
// If includeGroups is nil, {"album"} is assumed
func (spot *spotify) GetArtistDiscography(artist *db.Artist, includeGroups []db.AlbumType) ([]db.Album, error) {
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

	// collect ids of all albums in discography to slice
	albumIds := make([]string, len(response.Items))
	for albumIdx, album := range response.Items {
		albumIds[albumIdx] = album.Id
	}

	return spot.GetSeveralAlbumsById(albumIds)
}

// Given an Album, return all of its tracks. This method is necessary
// considering that sometimes db.Album objects won't have their .Tracks[]
// filled, for example if they came from db.Artist.FillDiscography(), etc...
// The Spotify Web API's /albums/<id>/tracks *does* return info about each track
// on the album, but in the form of a SimplifiedTrackObject. For that reason,
// here we perform two requests: one to collect Spotify IDs of all the tracks on the album
// and another to collect detailed info about each of the tracks using the dedicated
// /tracks endpoint and the spotify id(s) from the first request.
func (spot *spotify) GetAlbumTracklist(album *db.Album) ([]db.Track, error) {
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
