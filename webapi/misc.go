package webapi

import (
	"bool3max/musicdash/db"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Extract the db.User instance stored in the passed Gin context. This helper function is necessary as
// a simple gin.Context.Get("current_user") returns an "any" and an bool value, and it's cumbersome to have to do a type assertion every time.
// (user, _ := c.Get("current_user") and then: dbUser := user.(*db.User))
// If there is no User stored in the context, the function returns "nil". However, under normal circumstances this shouldn't
// ever happen as all handlers that would make use of this function are precedeed by the AuthNeeded middleware that makes
// sure that the request was properly authenticated, and if not, aborts the request.
func GetUserFromCtx(c *gin.Context) *db.User {
	userFromCtx, exists := c.Get("current_user")
	if !exists {
		return nil
	}

	return userFromCtx.(*db.User)
}

func AppendSpotifyBase64AuthCredentialsRequest(req *http.Request, client_id, client_secret string) {
	authCodeBase64 := base64.StdEncoding.EncodeToString([]byte(client_id + ":" + client_secret))
	req.Header.Add("Authorization", "Basic "+authCodeBase64)
}
