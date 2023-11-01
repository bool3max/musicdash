package main

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/service/spotify"
	"fmt"
	"log"
	"os"
)

func main() {
	var provider db.ResourceProvider
	provider, err := spotify.NewAPI(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_SECRET"))
	if err != nil {
		log.Fatal(err)
	}

	travis, _ := provider.GetTrackByMatch(db.ResourceIdentifier{Artist: "travis scott", Title: "90210"})
	fmt.Printf("%+v, %v", travis.Artists, len(travis.Artists))

	// plays, err := lastfm.GetAllPlays(os.Getenv("LASTFM_API_KEY"), "deadaptation")
	// if err != nil {
	// 	log.Fatal("failed from main", err)
	// }

	// for _, play := range plays {
	// 	fmt.Printf("%v - %v (%v)\n\t%v\n", play.Artist, play.Title, play.Album, play.Timestamp)
	// }

	// fmt.Printf("Total tracks: %v", len(plays))
}
