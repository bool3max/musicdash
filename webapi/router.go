package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/music"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func NewRouter(database *db.Db, spotifyProvider music.ResourceProvider) *gin.Engine {
	var router = gin.Default()

	api := router.Group("/api")
	{
		groupAccount := api.Group("/account")
		{

			// Sign-up using classic e-mail address and password combination.
			groupAccount.POST("/signup", HandlerSignupCred(database))

			// Log-in using e-mail address and password.
			groupAccount.POST("/login", HandlerLoginCred(database))

			// Obtain a Spotify authorization url that the user should be redirected to.
			// An url query parameter "flow_type" is required to be one of "connect" or "continue_with".
			groupAccount.GET(
				"/spotify-auth-url",
				HandlerSpotifyAuthUrl(database),
			)

			// Continuation of the "Continue with Spotify" flow. The client app sends a request to this endpoint
			// alongside the "code" and "state" parameters forwarded from Spotify.
			groupAccount.POST(
				"/spotify-continue-with",
				HandlerSpotifyContinueWith(database),
			)

			// from this point on all endpoints in the account group require valid auth
			groupAccount.Use(AuthNeeded(database))

			// Log out
			groupAccount.DELETE("/logout", HandlerLogout(database, false))

			// log out everywhere (i.e. revoke all active auth tokens for account)
			groupAccount.DELETE("/logout-all", HandlerLogout(database, true))

			// Link a Spotify account with an existing musicdash account. This endpoint requires the
			// "code" and "state" url query parameters to be forwarded from the spotify auth response.
			groupAccount.POST(
				"/spotify-link-account",
				HandlerSpotifyLinkAccount(database),
			)

			groupAccount.POST(
				"/upload-profile-image",
				HandlerUploadProfileImage(database),
			)

			groupAccount.POST(
				"/upload-profile-image/from-spotify",
				SpotifyAuthNeeded(database),
				HandlerUploadProfileImageFromSpotify(database),
			)

			groupAccount.POST(
				"/update-username",
				HandlerUpdateUsername(database),
			)
		}

		groupSpotify := api.Group("/spotify", AuthNeeded(database), SpotifyAuthNeeded(database))
		{
			// resourceType must be one of: album, playlist, artist. resourceId must be a valid id
			// for a resource of the corresponding resourceType
			// an optional URL parameter "count" may be supplied
			// the handler responds with a JSON-encoded array of URIs of all successfully queued tracks
			groupSpotify.POST("/random-queuer/:resourceType/:resourceId", HandlerRandomQueuer(database))

			groupSpotify.GET("/testing/currently-playing", func(c *gin.Context) {
				user := c.MustGet("current_user").(*db.User)
				spot := user.Spotify

				current, err := spot.GetCurrentlyPlayingInfo()
				if err != nil {
					log.Println(err)
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ERROR": "oops"})
				}

				fmt.Printf("trackId:%v\ntrackName:%v\nalbumName:%v\n", current.Track.SpotifyId, current.Track.Title, current.Track.Album.Title)

				c.Status(200)
			})
		}

		api.GET("/user/:userid/profile-image", HandlerGetUserProfileImage(database))
	}

	router.Use(static.ServeRoot("/", "./webapp"))

	return router
}
