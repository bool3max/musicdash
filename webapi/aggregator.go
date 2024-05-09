package webapi

import (
	"bool3max/musicdash/db"
	"context"
	"log"
	"time"
)

// An Aggregator is an object that's used to periodically aggregate registered users'
// track plays from Spotify and preserve them to the database
type Aggregator struct {
	db *db.Db
}

// Return a new Aggregator associated with a particular db.Db database.
func NewAggregator(database *db.Db) *Aggregator {
	return &Aggregator{
		db: database,
	}
}

// Run the aggregator, blocking the current goroutine and periodically fetching recent track plays for all
// musicdash users that have a linked Spotify account.
func (ag *Aggregator) Run() {
	for {
		log.Println("aggregator: performing run...")

		users, err := ag.db.GetUsersWithSpotifyLinked()
		if err != nil {
			log.Println("aggregator: fatal: error getting list of users: ", err)
			return
		}

		for _, user := range users {
			// obtain spotify client
			if err := user.GetSpotifyAuth(context.Background()); err != nil {
				log.Printf("error obtaining spotify client for user: {%v}: %v\n", user.Id.String(), err)
				continue
			}

			// use refreshedat column to calculate how many songs to request from spotify api
			var refreshedAt time.Time
			row := ag.db.Pool().QueryRow(
				context.Background(),
				`
					select refreshedat
					from auth.user
					where id=$1
				`,
				user.Id,
			)

			if err := row.Scan(&refreshedAt); err != nil {
				continue
			}

			countRequestTracks := 0

			// only ask the API for ask many 30-second songs the user's could've listened to
			// in time since the last refresh was performed. tiny optimization that saves us bandwidth and cpu cycles
			if time.Since(refreshedAt).Seconds() < (30 * 50) {
				countRequestTracks = int(time.Since(refreshedAt).Seconds() / 30)
			}

			plays, err := user.Spotify.GetRecentlyPlayedTracks(countRequestTracks)
			if err != nil {
				log.Printf("error getting recently played tracks for user: {%v}: %v\n", user.Id.String(), err)
			}
		}

		// Max. amount of time to sleep and not miss a single track play on Spotify
		time.Sleep(30 * 50 * time.Second)
	}
}
