package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/music"

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

		// groupSpotify := api.Group("/spotify")

		api.GET("/user/:userid/profile-image", HandlerGetUserProfileImage(database))
	}

	router.Use(static.ServeRoot("/", "./webapp"))

	return router
}
