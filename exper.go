package main

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/service/spotify"
	"fmt"
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

	// dbpool, err := db.NewPool()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	album, err := provider.GetAlbumByMatch(db.ResourceIdentifier{
		Artist: "Travis Scott",
		Title:  "UTOPIA",
	})

	if err != nil {
		log.Fatal("error getting travis", err)
	}

	fmt.Printf("%+v\n", album)

	// ----------

	// dbpool, err := db.NewPool()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// preserved, err := (&db.Artist{SpotifyId: "1234abcd"}).IsPreserved(context.Background(), dbpool)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// if preserved {
	// 	fmt.Println("track IN DATABASE")
	// } else {
	// 	fmt.Println("track NOT IN DATABASE")
	// }
}
