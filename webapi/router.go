package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/music"

	"github.com/gin-gonic/gin"
)

func NewRouter(database *db.Db, spotify music.ResourceProvider) *gin.Engine {
	var router = gin.Default()

	api := router.Group("/api")
	{
		groupAccount := api.Group("/account")
		{

			// Sign-up using classic e-mail address and password combination.
			groupAccount.POST("/signup_classic", HandlerSignupClassic(database))
			// Sign-up using an existing Spotify account.
			groupAccount.POST("/signup_spotify", func(c *gin.Context) {
				c.String(200, "aightr2")
			})
		}
	}

	return router
}
