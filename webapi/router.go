package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/music"
	"net/http"

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

			// Log out
			groupAccount.DELETE("/logout", AuthNeeded(database), HandlerLogout(database))
		}
	}

	return router
}
