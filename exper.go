package main

import (
	"bool3max/musicdash/db"
	"fmt"
	"log"
)

func main() {
	// spotify_client_id := os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID")
	// spotify_client_secret := os.Getenv("MUSICDASH_SPOTIFY_SECRET")

	db, err := db.New()
	if err != nil {
		log.Fatal("error initializing database", err)
	}

	album_id := "2uvBprdlMpzeN5Bq0PzMBI"
	album, err := db.GetTrackById(album_id)
	if err != nil {
		log.Fatal("error getting track", err)
	}

	fmt.Printf("track: %+v\n", album)
}
