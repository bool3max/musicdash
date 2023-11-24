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

	dummyDrake := db.Artist{SpotifyId: "3TVXtAsR1Inumwj472S9r4"}

	discog, err := dbobj.GetArtistDiscography(&dummyDrake, []db.AlbumType{db.AlbumSingle})
	if err != nil {
		log.Fatal("error getting drake discog", err)
	}

	fmt.Printf("%+v\n", discog)
}
