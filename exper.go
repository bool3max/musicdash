package main

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/spotify"
	"bool3max/musicdash/webapi"
	"context"
	"fmt"
	"log"
	"os"
)

func main() {
	database, err := db.Acquire()
	if err != nil {
		log.Fatal("error acquiring database object")
	}

	spot, err := spotify.NewAPI(os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID"), os.Getenv("MUSICDASH_SPOTIFY_SECRET"))
	if err != nil {
		log.Fatal("error acquiring spotify instance")
	}

	webapi.NewRouter(database, spot).Run(":7070")
}

func preserve_some_artists(artist_names []string) {
	spot, err := spotify.NewAPI(os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID"), os.Getenv("MUSICDASH_SPOTIFY_SECRET"))
	if err != nil {
		log.Fatal(err)
	}

	dbObj, err := db.Acquire()
	if err != nil {
		log.Fatal(err)
	}

	for _, artistName := range artist_names {
		fmt.Println("preserving ", artistName)
		artist, err := spot.GetArtistByMatch(artistName, 999, nil)
		if err != nil {
			fmt.Println("    failed fetching...")
			continue
		}

		if err := artist.Preserve(context.TODO(), dbObj.Pool(), true); err != nil {
			fmt.Println("   failed preserving to database...")
			fmt.Println(err)
		}
	}
}
