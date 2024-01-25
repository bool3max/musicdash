package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/music"
	"fmt"
	"net/http"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func NewRouter(database *db.Db, spotify music.ResourceProvider) *gin.Engine {
	var router = gin.Default()

	api := router.Group("/api")
	{
		groupAccount := api.Group("/account")
		{

			// Sign-up using classic e-mail address and password combination.
			groupAccount.POST("/signup", HandlerSignupCred(database))

			// Sign-up using an existing Spotify account.
			groupAccount.POST("/signup_spotify", func(c *gin.Context) {
				c.String(http.StatusNotImplemented, "not implemented")
			})

			// Log-in using e-mail address and password.
			groupAccount.POST("/login", HandlerLoginCred(database))

			// from this point on all endpoints in the account group require valid auth
			groupAccount.Use(AuthNeeded(database))

			// Log out
			groupAccount.DELETE("/logout", HandlerLogout(database, false))

			// log out everywhere (i.e. revoke all active auth tokens for account)
			groupAccount.DELETE("/logout_everywhere", HandlerLogout(database, true))

			// Initial endpoint in the process of connecting a Spotify account
			// that returns a URI to Spotify's account auth. API
			groupAccount.GET(
				"/spotify_connect",
				HandlerSpotifyConnectRedirect(database),
			)

			// Spotify's auth. api then redirects the user back to this endpoint
			groupAccount.GET(
				"/spotify_connect_callback",
				HandlerSpotifyConnectCallback(database),
			)
		}

		groupSpotify := api.Group("/spotify")
		groupSpotify.Use(AuthNeeded(database), SpotifyAuthNeeded(database))
		{
			groupSpotify.GET("/myProfile", func(c *gin.Context) {
				currentAccount := GetUserFromCtx(c)
				spot := currentAccount.Spotify

				accountDetails, err := spot.GetCurrentUserProfile()
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.String(http.StatusOK, fmt.Sprintf("%+v", accountDetails))
			})
		}

		api.GET("/res", AuthNeeded(database), func(ctx *gin.Context) {
			user := GetUserFromCtx(ctx)

			ctx.String(200, "you are logged in as: "+user.String())
		})
	}

	router.Use(static.ServeRoot("/", "./webapp"))

	return router
}
