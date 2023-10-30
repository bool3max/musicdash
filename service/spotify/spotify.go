// the spotify package deals with interaction with the Spotify api
package spotify

import (
	"bool3max/musicdash/db"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	endpointTrack  = "https://api.spotify.com/v1/tracks/"
	endpointArtist = "https://api.spotify.com/v1/artist/"
	endpointAlbum  = "https://api.spotify.com/v1/albums/"
	endpointSearch = "https://api.spotify.com/v1/search/"
)

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

type API struct {
	// authentication information of the current client
	auth spotifyAuth
}

// returns whether a refresh was attempted, and an error if the refresh failed or not
// if a refresh hasnt been attempted error is always nil
func (spot *API) validateToken() (bool, error) {
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
func NewSpotifyAPI(client_id, client_secret string) (*API, error) {
	spot := API{}

	spot.auth.client_id = client_id
	spot.auth.client_secret = client_secret

	err := spot.auth.refresh()
	if err != nil {
		return nil, err
	}

	return &spot, nil
}

func (spot *API) getHelper(uri string, decodeTo any) error {
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
// resourceType should be one of: "track", "album", "artist"
func (spot *API) getResource(resourceType string, iden Identifier) (db.Resource, error) {
	var resource db.Resource

	if iden.SpotifyID != "" {
		// spotify id of track present
		err := spot.getHelper(endpointTrack+iden.SpotifyID, &resource)
		if err != nil {
			return nil, err
		}
	} else {
		var searchResults Search
		searchQuery := url.Values{
			"q":     {url.QueryEscape(iden.Artist + " - " + iden.Title)},
			"limit": {"1"},
			"type":  {resourceType},
		}.Encode()
		err := spot.getHelper(endpointSearch+"?"+searchQuery, &searchResults)
		if err != nil {
			return nil, err
		}

		switch resourceType {
		case "track":
			if len(searchResults.Tracks.Items) < 1 {
				return nil, fmt.Errorf("no rersource found for %v", iden)
			}
			resource = &searchResults.Tracks.Items[0]
		case "artist":
			if len(searchResults.Artists.Items) < 1 {
				return nil, fmt.Errorf("no rersource found for %v", iden)
			}

			resource = &searchResults.Artists.Items[0]
		case "album":
			if len(searchResults.Albums.Items) < 1 {
				return nil, fmt.Errorf("no rersource found for %v", iden)
			}
			resource = &searchResults.Albums.Items[0]
		default:
			return nil, fmt.Errorf("unknown resource type: %v", resourceType)
		}

	}

	return resource, nil
}

func (spot *API) GetTrack(iden Identifier) (*db.Track, error) {
	resource, err := spot.getResource("track", iden)
	if err != nil {
		return nil, err
	}

	return resource.(*db.Track), nil
}

func (spot *API) GetAlbum(iden Identifier) (*db.Album, error) {
	resource, err := spot.getResource("album", iden)
	if err != nil {
		return nil, err
	}

	return resource.(*db.Album), nil
}

func (spot *API) GetArtist(iden Identifier) (*db.Artist, error) {
	resource, err := spot.getResource("artist", iden)
	if err != nil {
		return nil, err
	}

	return resource.(*db.Artist), nil
}

// passed to api functions when a resource is requested from Spotify's API
type Identifier struct {
	SpotifyID string
	Title     string
	Artist    string
}

type apiAccessToken struct {
	Access_token string `json:"access_token"`
	Token_type   string `json:"token_type"`
	Expires_in   int    `json:"expires_in"`
}

// a spotify resource (track, artist, album) that is able to be
// preserved in a database for later use

type Search struct {
	Tracks struct {
		Items []db.Track

		Href   string `json:"href"`
		Next   string `json:"next"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
		Total  int    `json:"total"`
	}

	Artists struct {
		Items []db.Artist

		Href   string `json:"href"`
		Next   string `json:"next"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
		Total  int    `json:"total"`
	}

	Albums struct {
		Items []db.Album

		Href   string `json:"href"`
		Next   string `json:"next"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
}
