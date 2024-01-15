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
		// /api/account endpoints are not hooked up to the authentication middleware
		// as they are used for account creation and logging in, before the user
		// has any credentials used for auth
		groupAccount := api.Group("/account")
		{

			// Sign-up using classic e-mail address and password combination.
			groupAccount.POST("/signup_cred", HandlerSignupCred(database))

			// Sign-up using an existing Spotify account.
			groupAccount.POST("/signup_spotify", func(c *gin.Context) {
				c.String(http.StatusNotImplemented, "not implemented")
			})

			// Log-in using e-mail address and password.
			groupAccount.POST("/login_cred", HandlerLoginCred(database))
		}
	}

	return router
}
