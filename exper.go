package main

import (
	"bool3max/musicdash/db"
	"fmt"
	"log"
)

func main() {
	// spotify_client_id := os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID")
	// spotify_client_secret := os.Getenv("MUSICDASH_SPOTIFY_SECRET")

	dbobj, err := db.New()
	if err != nil {
		log.Fatal("error initializing database", err)
	}

	artist, err := dbobj.GetArtistById("3TVXtAsR1Inumwj472S9r4", 2)
	if err != nil {
		log.Fatal("error getting drake", err)
	}

	for _, album := range artist.Discography {
		fmt.Printf("%s\n", album.Title)
		for _, track := range album.Tracks {
			fmt.Printf("\t%s\n", track.Title)
		}
	}
}
