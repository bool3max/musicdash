package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/music"
	"net/http"
	"os"

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
			groupAccount.DELETE("/logout", HandlerLogout(database))

			// Initial endpoint in connecting spotify account, which returns a redirect URI
			groupAccount.GET(
				"/spotify_connect",
				HandlerSpotifyConnectRedirect(database, os.Getenv("MUSICDASH_SPOTIFY_CLIENT_ID")),
			)
		}

		api.GET("/res", func(ctx *gin.Context) {
			ctx.String(200, "<h1>resource data</h1>")
		})
	}

	router.Use(static.ServeRoot("/", "./webapp"))

	return router
}
