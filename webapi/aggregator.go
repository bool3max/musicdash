package webapi

import (
	"bool3max/musicdash/db"
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

const AGGREGATOR_SLEEP_TIME = 30 * 50 * time.Second

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
			log.Printf("aggregator: processing user {%v}={%v}\n", user.Username, user.Id.String())
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
					select coalesce(refreshedat, '1900-01-01T00:00:01Z')
					from auth.user
					where id=$1
				`,
				user.Id,
			)

			if err := row.Scan(&refreshedAt); err != nil {
				log.Printf("error getting last refresh time: %v\n", err)
				continue
			}

			log.Printf("last refresh: %+v\n", refreshedAt)

			countRequestTracks := 0

			// only ask the API for ask many 30-second songs the user's could've listened to
			// in time since the last refresh was performed. tiny optimization that saves us bandwidth and cpu cycles
			if time.Since(refreshedAt).Seconds() < (30 * 50) {
				countRequestTracks = int(time.Since(refreshedAt).Seconds() / 30)
			}

			log.Printf("requesting %v tracks from spotify api...\n", countRequestTracks)

			playsNew, err := user.Spotify.GetRecentlyPlayedTracks(countRequestTracks)
			if err != nil {
				log.Printf("error getting recently played tracks for user: {%v}: %v\n", user.Id.String(), err)
				continue
			}

			// get most recent saved play from database
			playsOld, err := user.GetRecentPlaysFromDB(1, user.Spotify)
			if err != nil {
				log.Printf("error getting recently played tracks from db for user: {%v}: %v\n", user.Id.String(), err)
				continue
			}

			// special case - user has no recorded plays at all, save all plays for response
			if len(playsOld) == 0 {
				log.Println("user has no plays in db -> saving all plays from response...")
				if err := user.SavePlays(playsNew); err != nil {
					log.Printf("error saving new plays for user {%v}: %v\n", user.Id.String(), err)
				}

				// update user's refreshedat..
				_, err = ag.db.Pool().Exec(
					context.Background(),
					`
					update auth.user
					set refreshedat=@refreshedAt
					where userid=@userId	
				`,
					pgx.NamedArgs{
						"refreshedAt": time.Now(),
						"userId":      user.Id,
					},
				)

				if err != nil {
					log.Printf("error setting refreshed at for current user: %v\n", err)
					continue
				}

				continue
			}

			// otherwise, determine which new plays are never than most recent in db
			mostRecentDBPlay := playsOld[0]
			log.Printf("most recent play in db at: %+v\n", mostRecentDBPlay.At)
			var upperBoundIndex int
			for idx, newPlay := range playsNew {
				if newPlay.At.Before(mostRecentDBPlay.At) || newPlay.At.Equal(mostRecentDBPlay.At) {
					upperBoundIndex = idx
					break
				}

				// last play, still newer than most recent
				if idx == len(playsNew)-1 {
					upperBoundIndex = len(playsNew)
				}
			}

			// save all recent plays from the response that are newer than the most
			// recent play recorded in the database
			log.Printf("saving total of {%v} new plays...\n", len(playsNew[:upperBoundIndex]))
			if err := user.SavePlays(playsNew[:upperBoundIndex]); err != nil {
				log.Printf("error saving new plays for user {%v}: %v\n", user.Id.String(), err)
				continue
			}

			// update user's refreshedat..
			_, err = ag.db.Pool().Exec(
				context.Background(),
				`
					update auth.user
					set refreshedat=@refreshedAt
					where id=@userId	
				`,
				pgx.NamedArgs{
					"refreshedAt": time.Now(),
					"userId":      user.Id,
				},
			)

			if err != nil {
				log.Printf("error setting refreshed at for current user: %v\n", err)
				continue
			}
		}

		time.Sleep(5 * time.Minute)
	}
}
