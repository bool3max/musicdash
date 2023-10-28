// the lastfm package deals ONLY with fetching and exporting
// a lastfm user's COMPLETE list of all lifetime scrobbles
package lastfm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const apiEndpoint = "http://ws.audioscrobbler.com/2.0/"
const apiResponseMaxLimit = 200

type Play struct {
	Title     string
	Album     string
	Artist    string
	Timestamp time.Time
}

type recentTracks struct {
	Recenttracks struct {
		Track []struct {
			Artist struct {
				Name string `json:"#text"`
			}
			Album struct {
				Name string `json:"#text"`
			}

			Date struct {
				Uts string
			}

			Title string `json:"name"`
		}
		Attr struct {
			Total      int `json:"total,string"`
			TotalPages int `json:"totalPages,string"`
			Page       int `json:"page,string"`
			PerPage    int `json:"perPage,string"`
		} `json:"@attr"`
	} `json:"recenttracks"`
}

func GetAllPlays(api_key, username string) ([]Play, error) {
	query := url.Values{
		"method":  {"user.getrecenttracks"},
		"user":    {url.QueryEscape(username)},
		"format":  {"json"},
		"limit":   {strconv.Itoa(apiResponseMaxLimit)},
		"api_key": {api_key},
		"page":    {"1"},
	}

	req, err := http.NewRequest("GET", apiEndpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	var apiResponse recentTracks

	if err := json.NewDecoder(response.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	fmt.Printf("totalPages: %v, perPage: %v", apiResponse.Recenttracks.Attr.TotalPages, apiResponse.Recenttracks.Attr.PerPage)
	plays := make([]Play, 0, apiResponse.Recenttracks.Attr.Total)

	consumePlays := func(data *recentTracks) {
		for _, track := range data.Recenttracks.Track {
			unixtimetamp, err := strconv.Atoi(track.Date.Uts)
			if err != nil {
				continue
			}

			plays = append(plays, Play{
				Title:     track.Title,
				Artist:    track.Artist.Name,
				Album:     track.Album.Name,
				Timestamp: time.Unix(int64(unixtimetamp), 0),
			})
		}
	}

	// consume initial first page of results
	consumePlays(&apiResponse)

	if apiResponse.Recenttracks.Attr.TotalPages == 1 {
		// only one page of tracks available on account
		return plays, nil
	}

	// consume the rest of the pages
	for currentPageForRequest := 2; currentPageForRequest <= apiResponse.Recenttracks.Attr.TotalPages; currentPageForRequest++ {
		query.Set("page", strconv.Itoa(currentPageForRequest))

		req, err := http.NewRequest("GET", apiEndpoint+"?"+query.Encode(), nil)
		if err != nil {
			return nil, err
		}

		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		defer response.Body.Close()

		var apiResponse recentTracks

		if err := json.NewDecoder(response.Body).Decode(&apiResponse); err != nil {
			return nil, err
		}

		consumePlays(&apiResponse)
	}

	return plays, nil
}
