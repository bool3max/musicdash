package main

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/service/spotify"
	"context"
	"log"
	"os"
)

func main() {
	spotify_client_id := os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID")
	spotify_client_secret := os.Getenv("MUSICDASH_SPOTIFY_SECRET")

	provider, err := spotify.NewAPI(spotify_client_id, spotify_client_secret)
	if err != nil {
		log.Fatal("error creating spotify provider")
	}

	dbpool, err := db.NewPool()
	if err != nil {
		log.Fatal(err)
	}

	rodeo, err := provider.GetAlbumByMatch(db.ResourceIdentifier{
		Artist: "Travis Scott",
		Title:  "Rodeo",
	})

	if err != nil {
		log.Fatal("error getting travis", err)
	}

	preserved, err := rodeo.IsPreserved(context.Background(), dbpool)
	if err != nil {
		log.Fatal(err)
	}

	if !preserved {
		err = rodeo.Preserve(context.Background(), dbpool, true)
		if err != nil {
			log.Fatal(err)
		}
	}
}
