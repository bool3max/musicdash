package main

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/service/spotify"
	"context"
	"fmt"
	"log"
	"os"
)

var (
	spotify_client_id     = os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID")
	spotify_client_secret = os.Getenv("MUSICDASH_SPOTIFY_SECRET")
)

func main() {
	spot, err := spotify.NewAPI(spotify_client_id, spotify_client_secret)
	if err != nil {
		log.Fatal(err)
	}

	rolling_stones, err := spot.GetAlbumByMatch(db.ResourceIdentifier{Title: "lady gaga the fame monster"})
	if err != nil {
		log.Fatal(err)
	}

	dbpool, err := db.NewPool()
	if err != nil {
		log.Fatal(err)
	}

	tiniest := &rolling_stones.Images[len(rolling_stones.Images)-1]
	err = tiniest.Preserve(context.Background(), dbpool, false)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("preserved smallest")

}

// dumbass testing function
func preserve_some_artists() {
	pool, err := db.NewPool()
	if err != nil {
		log.Fatal(err)
	}

	spot, err := spotify.NewAPI(spotify_client_id, spotify_client_secret)
	if err != nil {
		log.Fatal(err)
	}

	for _, artistName := range []string{"Drake", "Lady Gaga", "Kanye West", "Ping Floyd"} {
		artist, err := spot.GetArtistByMatch(db.ResourceIdentifier{Artist: artistName}, 2)
		if err != nil {
			log.Fatal(err)
		}

		if err := artist.Preserve(context.Background(), pool, true); err != nil {
			log.Fatal("error preserving artist:", artistName, err)
		}
	}
}
