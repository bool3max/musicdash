package webapi

import (
	"bool3max/musicdash/db"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SignupCredRequestData struct {
	Username string `binding:"required"`
	Email    string `binding:"required"`
	Password string `binding:"required"`
}

type LoginCredRequestData struct {
	Email    string `binding:"required"`
	Password string `binding:"required"`
}

var (
	responseServerError  = gin.H{"message": "Server error."}
	responseBadRequest   = gin.H{"message": "Bad request."}
	responseNotLoggedIn  = gin.H{"message": "Not logged in."}
	responseInvalidLogin = gin.H{"message": "Invalid login."}
)

// Returns a Gin handler middleware that ensures that the user is logged-in into a valid
// session. If the user isn't logged in or the token is invalid, this middleware aborts the
// handler chain (if that's the right term for it?)
func AuthNeeded(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken, err := c.Cookie("auth_token")
		// no auth session cookie
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseNotLoggedIn)
			return
		}

		var userId uuid.UUID

		err = database.Pool().QueryRow(
			c,
			`
				select userid
				from auth.auth_token	
				where auth.auth_token.token=$1
			`,
			authToken,
		).Scan(&userId)

		if err != nil {
			log.Println(err)
			// auth token not in database
			if err == pgx.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusUnauthorized, responseInvalidLogin)
				return
			}

			// other db error
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseServerError)
		}
	}
}

// Gin handler for signing up using an email and password.
func HandlerSignupCred(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data SignupCredRequestData
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		if len(data.Username) < 3 || len(data.Username) > 30 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"message": "Username length must be between 3 and 30 characters."},
			)
			return
		}

		exists, err := database.UsernameIsRegistered(c, data.Username)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseServerError)
			return
		}

		if exists {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{"message": "An account with that username already exists."},
			)
			return
		}

		// bcrypt max password byte length is 72 bytes and we salt it with a uuidv4 which is 16 bytes
		if len(data.Password) < 8 || len(data.Password) > (72-16) {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{"message": "Password length must be at least 8 characters and no more than 56 characters."},
			)
			return
		}

		_, err = mail.ParseAddress(data.Email)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"message": "Invalid e-mail address."},
			)
			return
		}

		if err = database.UserInsert(data.Username, data.Password, data.Email); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseServerError)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Account created successfully."})
	}
}

func HandlerLoginCred(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data LoginCredRequestData
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		// validate e-mail address
		_, err := mail.ParseAddress(data.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid e-mail address."})
			return
		}

		authToken, err := database.UserLogin(c, data.Password, data.Email)
		if err != nil {
			if err == db.ErrorEmailNotRegistered || err == db.ErrorPasswordIncorrect {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Incorrect login credentials."})
				return
			}

			c.JSON(http.StatusInternalServerError, responseServerError)
			return
		}

		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("auth_token", string(authToken), int((time.Hour * 24 * 30).Seconds()), "/", "", true, true)
		c.JSON(http.StatusOK, gin.H{"token": authToken})
	}
}

func HandlerLogout(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		// at point of this handler being called we know the user is logged into a valid sessin
		// because of the preceding auth middleware

		authToken, err := c.Cookie("auth_token")
		if err != nil {
			c.JSON(http.StatusInternalServerError, responseServerError)
			return
		}

		err = database.UserRevokeToken(
			c,
			db.UserAuthToken(authToken),
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, responseServerError)
			return
		}

		// instruct client to clear cookie
		c.SetCookie("auth_token", "", -1, "/", "", true, true)
		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully."})
	}
}

func HandlerSpotifyConnectRedirect(database *db.Db, spotify_client_id string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// generate random string for state parameter
		var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

		state := make([]rune, 32)
		for i := range state {
			state[i] = letters[rand.Intn(len(letters))]
		}

		endpoint := "https://api.spotify.com/v1/authorize"
		params := url.Values{
			"client_id":     {spotify_client_id},
			"response_type": {"code"},
			"redirect_uri":  {"http://localhost:7070/api/spotify_connect_callback"},
			"scope":         {"user-read-playback-position user-top-read user-read-recently-played user-library-read user-read-playback-state user-modify-playback-state user-read-currently-playing"},
			"state":         {string(state)},
		}

		final := endpoint + "?" + params.Encode()

		// save the generated random state on the client
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("spotify_connect_state", string(state), 300, "/", "", true, true)

		c.JSON(http.StatusOK, gin.H{"redirect_url": final})
	}
}
