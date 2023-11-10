package main

import (
	"bool3max/musicdash/db"
	"context"
	"fmt"
	"log"
)

func main() {
	// var provider db.ResourceProvider
	// provider, err := spotify.NewAPI(os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID"), os.Getenv("MUSICDASH_SPOTIFY_SECRET"))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// album, err := provider.GetAlbumByMatch(db.ResourceIdentifier{Artist: "travis scott", Title: "rodeo"})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fullAlbum, err := provider.GetAlbumById(album.SpotifyId)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// for _, track := range fullAlbum.Tracks {
	// 	fmt.Printf("%v -- {%v}\n", track.Title, track.Duration)
	// }

	drake := db.Artist{
		SpotifyId: "1234abcd",
	}

	dbpool, err := db.NewPool()
	if err != nil {
		log.Fatal(err)
	}

	preserved, err := drake.IsPreserved(context.Background(), dbpool)
	if err != nil {
		log.Fatal(err)
	}

	if preserved {
		fmt.Println("DRAKE IN DATABASE")
	} else {
		fmt.Println("DRAKE NOT IN DATABASE")
	}

	// plays, err := lastfm.GetAllPlays(os.Getenv("LASTFM_API_KEY"), "deadaptation")
	// if err != nil {
	// 	log.Fatal("failed from main", err)
	// }

	// for _, play := range plays {
	// 	fmt.Printf("%v - %v (%v)\n\t%v\n", play.Artist, play.Title, play.Album, play.Timestamp)
	// }

	// fmt.Printf("Total tracks: %v", len(plays))
}
