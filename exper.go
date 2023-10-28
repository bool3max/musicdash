package main

import (
	"bool3max/musicdash/service/spotify"
	"fmt"
	"log"
	"os"
)

func main() {
	spot, err := spotify.NewSpotifyAPI(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_SECRET"))
	if err != nil {
		log.Fatal(err)
	}

	track, err := spot.GetTrack(spotify.Identifier{Title: "90210", Artist: "travis scott"})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(track)

	// plays, err := lastfm.GetAllPlays(os.Getenv("LASTFM_API_KEY"), "deadaptation")
	// if err != nil {
	// 	log.Fatal("failed from main", err)
	// }

	// for _, play := range plays {
	// 	fmt.Printf("%v - %v (%v)\n\t%v\n", play.Artist, play.Title, play.Album, play.Timestamp)
	// }

	// fmt.Printf("Total tracks: %v", len(plays))
}
