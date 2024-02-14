package spotify

import (
	music "bool3max/musicdash/music"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxIdsGetSeveralAlbums = 20

const (
	endpointTrack  = "https://api.spotify.com/v1/tracks/"
	endpointArtist = "https://api.spotify.com/v1/artists/"
	endpointAlbum  = "https://api.spotify.com/v1/albums/"
	endpointSearch = "https://api.spotify.com/v1/search/"
	endpointToken  = "https://accounts.spotify.com/api/token"
)

// Type of authentication flow used to instantiate a new client
type AuthFlowType int

// only these two are supported so far
const (
	ClientCredentials = iota
	AuthorizationCode = iota
)

var (
	// returned by Client.jsonGetHelper when there is no body in http response to JSON-decode
	ErrNoBody                    = errors.New("empty body in http response")
	ErrUserNotPlaying            = errors.New("user not playing anything")
	ErrInvalidAuthFlowForRequest = errors.New("invalid client authentication flow for request")
)

// An authenticated client used to interact with the API
type Client struct {
	// type of flow used to authenticate current client
	flowType AuthFlowType
	// the Client ID and Client secret of the spotify api app used to instantiate the client
	Client_id, Client_secret string

	// a base64 string encoding of "<client_id>:<client_secret>"
	clientPairBase64 string

	// the access token used to make requests to the api
	AccessToken string

	// the precise point in time at which the access_token becomes invalid
	ExpiresAt time.Time

	// the refresh token used to obtain new access tokens once they expire (only valid for AuthorizationCode, otherwise "")
	RefreshToken string

	// available scopes (only when using AuthorizationCode auth flow, otherwise empty string)
	Scope string
}

func base64EncodeClientPair(client_id, client_secret string) string {
	return base64.StdEncoding.EncodeToString([]byte(client_id + ":" + client_secret))
}

func NewClientCredentials(client_id, client_secret string) (*Client, error) {
	newClient := &Client{
		flowType:         ClientCredentials,
		Client_id:        client_id,
		Client_secret:    client_secret,
		clientPairBase64: base64EncodeClientPair(client_id, client_secret),
	}

	// for the client credentials auth flow obtaining a new access token is the same process
	// as refreshing it, so we use that same method when constructing a new client

	_, err := newClient.Refresh()
	if err != nil {
		return nil, err
	}

	return newClient, nil
}

// Create and return a new *Client from provided parameters. No error checking for their validity is done.
func AuthorizationCodeFromParams(client_id, client_secret, access_token, refresh_token string, expires_at time.Time) *Client {
	return &Client{
		flowType:         AuthorizationCode,
		Client_id:        client_id,
		Client_secret:    client_secret,
		clientPairBase64: base64EncodeClientPair(client_id, client_secret),
		AccessToken:      access_token,
		RefreshToken:     refresh_token,
		ExpiresAt:        expires_at,
	}
}

// The "redirect_uri" parameter is necessary as, when exchanging the "code" url query parameter for an authentication
// token, a redirect_uri in the body must also be supplied that has previously been used to obtain the "code" in the
// first place. I guess this is used for security on Spotify's end for some reason.
func NewAuthorizationCode(client_id, client_secret, code, redirect_uri string) (*Client, error) {
	newClient := &Client{
		flowType:         AuthorizationCode,
		Client_id:        client_id,
		Client_secret:    client_secret,
		clientPairBase64: base64EncodeClientPair(client_id, client_secret),
	}

	bodyData := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirect_uri},
	}.Encode()

	req, err := http.NewRequest("POST", endpointToken, strings.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic "+newClient.clientPairBase64)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error authenticating acess token for authorization code: spotify status code: %d", resp.StatusCode)
	}

	var response struct {
		Access_token  string `json:"access_token"`
		Token_type    string `json:"token_type"`
		Scope         string `json:"scope"`
		Expires_in    int    `json:"expires_in"`
		Refresh_token string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	newClient.AccessToken = response.Access_token
	newClient.RefreshToken = response.Refresh_token
	newClient.ExpiresAt = time.Now().Add(time.Duration(response.Expires_in) * time.Second)
	newClient.Scope = response.Scope

	return newClient, nil
}

// The boolean indicates if a refresh was attempted or not, and the error indicates if it succeeded or not.
// If a refresh hasn't been attempted, error is nil.
func (client *Client) Refresh() (bool, error) {
	// existing access token that is still valid, no need to refresh
	if client.AccessToken != "" && time.Now().Before(client.ExpiresAt) {
		return false, nil
	}

	if client.flowType == ClientCredentials {
		bodyData := url.Values{
			"grant_type": {"client_credentials"},
		}.Encode()

		req, err := http.NewRequest("POST", endpointToken, strings.NewReader(bodyData))
		if err != nil {
			return true, err
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Authorization", "Basic "+client.clientPairBase64)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return true, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return true, fmt.Errorf("error refreshing access token for client credentials: spotify status code: %d", resp.StatusCode)
		}

		var response struct {
			Access_token string `json:"access_token"`
			Token_type   string `json:"token_type"`
			Expires_in   int    `json:"expires_in"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return true, err
		}

		client.AccessToken = response.Access_token
		client.ExpiresAt = time.Now().Add(time.Duration(response.Expires_in) * time.Second)

		return true, nil

	} else if client.flowType == AuthorizationCode {
		bodyData := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {client.RefreshToken},
		}.Encode()

		req, err := http.NewRequest("POST", endpointToken, strings.NewReader(bodyData))
		if err != nil {
			return true, err
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Authorization", "Basic "+client.clientPairBase64)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return true, err
		}

		defer resp.Body.Close()

		var response struct {
			Access_token  string `json:"access_token"`
			Token_type    string `json:"token_type"`
			Scope         string `json:"scope"`
			Expires_in    int    `json:"expires_in"`
			Refresh_token string `json:"refresh_token"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return true, err
		}

		if resp.StatusCode != 200 {
			return true, fmt.Errorf("error refreshing access token for authorization code: spotify status code: %d", resp.StatusCode)
		}

		client.AccessToken = response.Access_token
		if response.Refresh_token != "" {
			client.RefreshToken = response.Refresh_token
		}

		client.ExpiresAt = time.Now().Add(time.Duration(response.Expires_in) * time.Second)

		return true, nil
	} else {
		return true, errors.New("unsuppored client flow type")
	}
}

// helper function that performs a GET request to the specified uri with an appended Authorization
// header and decodes the body as JSON to the specified destination
func (client *Client) jsonGetHelper(uri string, decodeTo any) (int, error) {
	req, err := http.NewRequest("GET", uri, nil)

	// error constructing request
	if err != nil {
		return -1, err
	}

	req.Header.Add("Authorization", "Bearer "+client.AccessToken)

	response, err := http.DefaultClient.Do(req)

	// error performing request
	if err != nil {
		return -1, err
	}

	defer response.Body.Close()

	// no content in body to decode
	if response.ContentLength != -1 && response.ContentLength <= 0 {
		return response.StatusCode, ErrNoBody
	}

	// decode body to destination
	if err := json.NewDecoder(response.Body).Decode(decodeTo); err != nil {
		return response.StatusCode, err
	}

	return response.StatusCode, nil
}

func (spot *Client) Search(query string, limit int) (Search, error) {
	searchQuery := url.Values{
		"q":     {url.QueryEscape(query)},
		"limit": {strconv.Itoa(limit)},
		"type":  {"album,track,artist"},
	}.Encode()

	var searchResults searchResponse
	_, err := spot.jsonGetHelper(endpointSearch+"?"+searchQuery, &searchResults)

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

func (spot *Client) GetTrackById(id string) (*music.Track, error) {
	var track track
	if _, err := spot.jsonGetHelper(endpointTrack+id, &track); err != nil {
		return nil, err
	}

	dbTrack := track.toDB()

	return &dbTrack, nil
}

func (spot *Client) GetSeveralTracksById(ids []string) ([]music.Track, error) {
	requestUrl := endpointTrack + "?" + url.Values{"ids": {strings.Join(ids, ",")}}.Encode()
	var result struct {
		Tracks []track `json:"tracks"`
	}
	_, err := spot.jsonGetHelper(requestUrl, &result)
	if err != nil {
		return nil, err
	}

	dbTracks := make([]music.Track, len(ids))
	for trackIdx, track := range result.Tracks {
		dbTracks[trackIdx] = track.toDB()
	}

	return dbTracks, nil
}

func (spot *Client) GetTrackByMatch(iden string) (*music.Track, error) {
	// perform a search to obtain id of the desired track
	search, err := spot.Search(iden, 1)
	if err != nil {
		return nil, err
	}

	firstResultId := search.Tracks[0].SpotifyId

	return spot.GetTrackById(firstResultId)
}

func (spot *Client) GetArtistById(id string, discogFillLevel int, albumTypes []music.AlbumType) (*music.Artist, error) {
	var artist artist
	if _, err := spot.jsonGetHelper(endpointArtist+id, &artist); err != nil {
		return nil, err
	}

	dbArtist := artist.toDB()

	if discogFillLevel > 0 {
		if err := dbArtist.FillDiscography(spot, albumTypes, discogFillLevel > 1); err != nil {
			return nil, err
		}
	}

	return &dbArtist, nil
}

func (spot *Client) GetArtistByMatch(iden string, discogFillLevel int, albumTypes []music.AlbumType) (*music.Artist, error) {
	search, err := spot.Search(iden, 1)
	if err != nil {
		return nil, err
	}

	firstResultId := search.Artists[0].SpotifyId
	artist, err := spot.GetArtistById(firstResultId, discogFillLevel, albumTypes)
	if err != nil {
		return nil, err
	}

	return artist, nil
}

func (spot *Client) GetAlbumById(id string) (*music.Album, error) {
	var album album
	if _, err := spot.jsonGetHelper(endpointAlbum+id, &album); err != nil {
		return nil, err
	}

	dbAlbum := album.toDB()

	return &dbAlbum, nil
}

func (spot *Client) GetSeveralAlbumsById(ids []string) ([]music.Album, error) {
	// if number of ids requested is more than allowed by api, split slice into
	// chunks of max allowed size
	if len(ids) > maxIdsGetSeveralAlbums {
		all := make([]music.Album, 0, len(ids))
		for startIdx := 0; startIdx < len(ids); startIdx += maxIdsGetSeveralAlbums {
			var idsRange []string

			if startIdx+maxIdsGetSeveralAlbums > len(ids) {
				idsRange = ids[startIdx:]
			} else {
				idsRange = ids[startIdx : startIdx+maxIdsGetSeveralAlbums]
			}

			batch, err := spot.GetSeveralAlbumsById(idsRange)
			if err != nil {
				return nil, err
			}

			all = append(all, batch...)
		}

		return all, nil
	}

	requestUrl := endpointAlbum + "?" + url.Values{"ids": {strings.Join(ids, ",")}}.Encode()
	var result struct {
		Albums []album `json:"albums"`
	}
	_, err := spot.jsonGetHelper(requestUrl, &result)
	if err != nil {
		return nil, err
	}

	dbAlbums := make([]music.Album, len(ids))
	for albumIdx, album := range result.Albums {
		dbAlbums[albumIdx] = album.toDB()
	}

	return dbAlbums, nil
}

func (spot *Client) GetAlbumByMatch(iden string) (*music.Album, error) {
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
func (spot *Client) GetArtistDiscography(artist *music.Artist, includeGroups []music.AlbumType) ([]music.Album, error) {
	if includeGroups == nil {
		includeGroups = []music.AlbumType{music.AlbumRegular}
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
		"include_groups": {music.IncludeGroupToString(includeGroups)},
	}.Encode()

	_, err := spot.jsonGetHelper(endpointArtist+artist.SpotifyId+"/albums?"+queryParams, &response)
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
func (spot *Client) GetAlbumTracklist(album *music.Album) ([]music.Track, error) {
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

	_, err := spot.jsonGetHelper(endpointAlbum+album.SpotifyId+"/tracks?"+queryParams, &response)
	if err != nil {
		return nil, err
	}

	tracklistIds := make([]string, len(response.Items))
	for idx, track := range response.Items {
		tracklistIds[idx] = track.Id
	}

	albumTracks, err := spot.GetSeveralTracksById(tracklistIds)
	if err != nil {
		return []music.Track{}, err
	}

	return albumTracks, nil
}

func (spot *Client) GetCurrentUserProfile() (UserProfile, error) {
	if spot.flowType != AuthorizationCode {
		return UserProfile{}, ErrInvalidAuthFlowForRequest
	}

	var response struct {
		Id          string `json:"id"`
		Country     string `json:"country"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Followers   struct {
			Total int `json:"total"`
		} `json:"followers"`
		Images       []image `json:"images"`
		Uri          string  `json:"uri"`
		ExternalUrls struct {
			Spotify string `json:"spotify"`
		} `json:"external_urls"`
	}

	if statusCode, err := spot.jsonGetHelper("https://api.spotify.com/v1/me", &response); err != nil {
		log.Println("error getting profile, status: ", statusCode)
		return UserProfile{}, err
	}

	newUserProfile := UserProfile{
		SpotifyId:     response.Id,
		DisplayName:   response.DisplayName,
		FollowerCount: uint(response.Followers.Total),
		ProfileUri:    response.Uri,
		ProfileUrl:    response.ExternalUrls.Spotify,
		ProfileImages: make([]music.Image, len(response.Images)),
		Country:       response.Country,
		Email:         response.Email,
	}

	for idx, img := range response.Images {
		newUserProfile.ProfileImages[idx] = music.Image{
			Height: int(img.Height),
			Width:  int(img.Width),
			Url:    img.Url,
		}
	}

	return newUserProfile, nil
}

func (spot *Client) GetCurrentlyPlayingInfo() (CurrentlyPlaying, error) {
	if spot.flowType != AuthorizationCode {
		return CurrentlyPlaying{}, ErrInvalidAuthFlowForRequest
	}

	var response struct {
		IsPlaying  bool `json:"is_playing"`
		ProgressMs int  `json:"progress_ms"`
		Context    struct {
			Type         string `json:"type"`
			Href         string `json:"href"`
			ExternalUrls struct {
				Spotify string `json:"spotify"`
			} `json:"external_urls"`
		} `json:"context"`
		Item                 track  `json:"item"`
		CurrentlyPlayingType string `json:"currently_playing_type"`
	}

	if statusCode, err := spot.jsonGetHelper("https://api.spotify.com/v1/me/player/currently-playing", &response); err != nil {
		// no response and 204 -> user is not playing anything
		if err == ErrNoBody && statusCode == http.StatusNoContent {
			return CurrentlyPlaying{}, ErrUserNotPlaying
		}

		// different error
		return CurrentlyPlaying{}, err
	}

	// playing podcast episode or ad
	if response.CurrentlyPlayingType != "track" {
		return CurrentlyPlaying{}, ErrUserNotPlaying
	}

	return CurrentlyPlaying{
		IsPlayingNow: response.IsPlaying,
		Track:        response.Item.toDB(),
		Progress:     time.Duration(response.ProgressMs * int(time.Millisecond)),
	}, nil
}

func (spot *Client) GetRecentlyPlayedTracks() ([]music.Play, error) {
	if spot.flowType != AuthorizationCode {
		return nil, ErrInvalidAuthFlowForRequest
	}

	var response struct {
		Total int    `json:"total"`
		Next  string `json:"next"`
		Items []struct {
			Track    track     `json:"track"`
			PlayedAt time.Time `json:"played_at"`
		} `json:"items"`
	}

	if _, err := spot.jsonGetHelper("https://api.spotify.com/v1/me/player/recently-played", &response); err != nil {
		// no response and 204 -> user is not playing anything
		return nil, err
	}

	convertedPlays := make([]music.Play, response.Total)
	for idx, play := range response.Items {
		convertedPlays[idx] = music.Play{
			At:    play.PlayedAt,
			Track: play.Track.toDB(),
		}
	}

	return convertedPlays, nil
}
