package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/music"
	"bool3max/musicdash/spotify"
	"fmt"
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
				"/spotify_auth_url",
				HandlerSpotifyAuthUrl(database),
			)

			// Continuation of the "Continue with Spotify" flow. The client app sends a request to this endpoint
			// alongside the "code" and "state" parameters forwarded from Spotify.
			groupAccount.POST(
				"/spotify_continue_with",
				HandlerSpotifyContinueWith(database),
			)

			// from this point on all endpoints in the account group require valid auth
			groupAccount.Use(AuthNeeded(database))

			// Log out
			groupAccount.DELETE("/logout", HandlerLogout(database, false))

			// log out everywhere (i.e. revoke all active auth tokens for account)
			groupAccount.DELETE("/logout_everywhere", HandlerLogout(database, true))

			// Link a Spotify account with an existing musicdash account. This endpoint requires the
			// "code" and "state" url query parameters to be forwarded from the spotify auth response.
			groupAccount.POST(
				"/spotify_link_account",
				HandlerSpotifyLinkAccount(database),
			)
		}

		groupSpotify := api.Group("/spotify")
		groupSpotify.Use(AuthNeeded(database), SpotifyAuthNeeded(database))
		{
			groupSpotify.GET("/currentlyPlaying", func(c *gin.Context) {
				currentAccount := GetUserFromCtx(c)
				spot := currentAccount.Spotify

				currentlyPlaying, err := spot.GetCurrentlyPlayingInfo()
				if err != nil {
					if err == spotify.ErrUserNotPlaying {
						c.JSON(http.StatusOK, gin.H{"error": "ERROR_USER_NOT_PLAYING"})
						return
					}

					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "SPOTIFY"})
					return
				}

				c.JSON(http.StatusOK, currentlyPlaying)
			})
		}

		api.GET("/res", AuthNeeded(database), func(ctx *gin.Context) {
			user := GetUserFromCtx(ctx)

			ctx.String(200, fmt.Sprintf("You are logged in as: %+v\n", *user))
		})
	}

	router.Use(static.ServeRoot("/", "./webapp"))

	return router
}
