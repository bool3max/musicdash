package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/spotify"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/h2non/bimg"
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
	responseInternalServerError = gin.H{"error": "ERROR_INTERNAL_SERVER"}
	responseBadRequest          = gin.H{"error": "ERROR_BAD_RQUEST"}
	responseNotLoggedIn         = gin.H{"error": "ERROR_NOT_LOGGED_IN"}
	responseInvalidLogin        = gin.H{"error": "ERROR_INVALID_LOGIN"}
)

// Returns a Gin handler middleware that ensures that the user is logged-in into a valid
// session. If the user isn't logged in or the token is invalid, this middleware aborts the
// handler chain (if that's the right term for it?). Otherwise it saves the current logged in user
// as into the gin context.
func AuthNeeded(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken, err := c.Cookie("auth_token")
		// no auth session cookie present
		if err != nil {
			log.Println("no auth cookie present: ", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, responseNotLoggedIn)
			return
		}

		user, err := database.GetUserFromAuthToken(c, db.UserAuthToken(authToken))
		if err != nil {
			if err == db.ErrInvalidAuthToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized, responseInvalidLogin)
				return
			}

			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		// save the auth token used and the User into the gin context for future handlers to make use of
		c.Set("current_auth_token", authToken)
		c.Set("current_user", &user) // the user instance is saved as a pointer
	}
}

// Returns a Gin handler middleware that ensures that the user performing the current request
// has a connected Spotify account that is currently properly authenticated. As such, this middleware
// must be preceeded by the AuthNeeded middleware. If the user has a connected Spotify account
// but it is currently not authenticated (i.e. the access token is expired), the middleware
// attempts to refresh it. Upon successfull validation, the middleware attaches an instance of
// db.SpotifyAuthParams to the existing db.User value in the current context.
func SpotifyAuthNeeded(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUserFromCtx(c)

		var accessToken, refreshToken string
		var expiresAt time.Time

		err := database.Pool().QueryRow(
			c,
			`
				select accesstoken, refreshtoken, expiresat
				from auth.spotify_token
				where userid=$1
			`,
			user.Id,
		).Scan(&accessToken, &refreshToken, &expiresAt)

		if err != nil {
			if err == pgx.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "ERROR_SPOTIFY_UNAUTHENTICATED"})
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			}

			return
		}

		userSpotifyClient := spotify.AuthorizationCodeFromParams(
			db.MUSICDASH_SPOTIFY_CLIENT_ID,
			db.MUSICDASH_SPOTIFY_SECRET,
			accessToken,
			refreshToken,
			expiresAt,
		)

		// refresh access token
		if _, err = userSpotifyClient.Refresh(); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SPOTIFY_AUTHORIZATION"})
			return
		}

		// save spotify client instance to db.User instance for future handlers to make use of
		user.Spotify = userSpotifyClient
		// save potentially-refreshed parameters of client to database
		if err = user.SaveSpotifyAuthParams(c); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
			return
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

		// trim all leading and trailing whitespace from the username
		data.Username = strings.TrimSpace(data.Username)

		// check that the username is valid
		if !UsernameIsValid(data.Username) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"message": "Invalid username."},
			)
			return
		}

		exists, err := database.UsernameIsRegistered(c, data.Username)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
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

		if _, err = database.UserInsert(data.Username, data.Password, data.Email); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
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

		// validate login credentials and obtain userId of user
		var userId uuid.UUID
		if userId, err = database.UserValidateLoginCred(c, data.Password, data.Email); err != nil {
			if err == db.ErrEmailNotRegistered || err == db.ErrPasswordIncorrect {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Incorrect login credentials."})
				return
			}

			c.JSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		// login credentials valid, obtain auth token of requested user
		var authToken db.UserAuthToken
		authToken, err = database.UserNewAuthToken(c, userId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", string(authToken), int((time.Hour * 24 * 30).Seconds()), "/", "", true, true)
		c.JSON(http.StatusOK, gin.H{"token": authToken})
	}
}

func HandlerLogout(database *db.Db, everywhere bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken := c.GetString("current_auth_token")

		if everywhere {
			// log out everywhere
			user := GetUserFromCtx(c)
			err := user.RevokeAllTokens(c)

			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
				return
			}
		} else {
			// log out just this auth token
			err := database.RevokeAuthToken(
				c,
				db.UserAuthToken(authToken),
			)

			if err != nil {
				c.JSON(http.StatusInternalServerError, responseInternalServerError)
				return
			}
		}

		// instruct client to clear cookie
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", "", -1, "/", "", true, true)

		// clear login auth info from context
		c.Set("current_user", nil)
		c.Set("current_auth_token", nil)

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully."})
	}
}

// This API endpoint returns a new Spotify auth redirect URL that the user's frontend is redirected to
// in order to perform authorization with spotify.
func HandlerSpotifyAuthUrl(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		// flow_type must be one of "connect" or "continue_with"

		flowType := c.Query("flow_type")
		if flowType == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ERROR_MISSING_FLOW_TYPE"})
			return
		}

		// generate random string for state parameter
		var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

		state := make([]rune, 32)
		for i := range state {
			state[i] = letters[rand.Intn(len(letters))]
		}

		var spotifyRedirectUri = "http://localhost:7070/"
		switch flowType {
		case "connect":
			spotifyRedirectUri += "#spotify_connect_account"
		case "continue_with":
			spotifyRedirectUri += "#spotify_continue_with"
		default:
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ERROR_INVALID_FLOW_TYPE"})
			return
		}

		endpoint := "https://accounts.spotify.com/authorize"
		params := url.Values{
			"client_id":     {db.MUSICDASH_SPOTIFY_CLIENT_ID},
			"response_type": {"code"},
			"redirect_uri":  {spotifyRedirectUri},
			"scope":         {"user-read-playback-position user-top-read user-read-recently-played user-library-read user-read-playback-state user-modify-playback-state user-read-currently-playing user-read-email user-read-private"},
			"state":         {string(state)},
		}

		final := endpoint + "?" + params.Encode()

		// save the generated random state on the client
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("spotify_connect_state", string(state), 300, "/", "", true, true)

		c.JSON(http.StatusOK, gin.H{"redirect_url": final})
	}
}

// The frontend makes a request to this handler once the user successfully authorizes with spotify.
// Spotify redirects the user back to the app and provides the "state" and "code" parameters which
// are then forwarded to this api handler which then authorizes a new spotify client and links
// it to the user's musicdash account.
func HandlerSpotifyLinkAccount(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUserFromCtx(c)

		queryState := c.Query("state")
		queryCode := c.Query("code")
		queryErr := c.Query("error")

		clientState, err := c.Cookie("spotify_connect_state")

		if queryState == "" || queryCode == "" || err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		if queryErr != "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "spotify error: " + queryErr})
			return
		}

		if queryState != clientState {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "States don't match."})
			return
		}

		// authenticate new spotify.Client using auth. code flow with the code response
		userSpotifyClient, err := spotify.NewAuthorizationCode(
			db.MUSICDASH_SPOTIFY_CLIENT_ID,
			db.MUSICDASH_SPOTIFY_SECRET,
			queryCode,
			"http://localhost:7070/#spotify_connect_account",
		)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "ERROR_SPOTIFY_AUTHORIZATION"})
			return
		}

		// successfully authenticated with spotify
		user.Spotify = userSpotifyClient

		spotifyProfile, err := user.Spotify.GetCurrentUserProfile()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		if err = user.LinkSpotifyProfile(c, spotifyProfile); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		// Preserve the spotify auth params into the database. If linking spotify acc fails
		// we don't want to store auth credentials.
		if err := user.SaveSpotifyAuthParams(c); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}
	}
}

func HandlerSpotifyContinueWith(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		queryState := c.Query("state")
		queryCode := c.Query("code")
		queryError := c.Query("error")

		clientState, err := c.Cookie("spotify_connect_state")

		if queryState == "" || queryCode == "" || err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		if queryError != "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "spotify error: " + queryError})
			return
		}

		if queryState != clientState {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "states don't match."})
			return
		}

		// authenticate new spotify.Client using auth. code flow with the code response
		userSpotifyClient, err := spotify.NewAuthorizationCode(
			db.MUSICDASH_SPOTIFY_CLIENT_ID,
			db.MUSICDASH_SPOTIFY_SECRET,
			queryCode,
			"http://localhost:7070/#spotify_continue_with",
		)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "ERROR_SPOTIFY_AUTHORIZATION"})
			return
		}

		spotifyProfile, err := userSpotifyClient.GetCurrentUserProfile()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "error getting spotify user profile"})
			return
		}

		var eventualAuthToken db.UserAuthToken

		// check for an existing musicdash account with same linked spotify account that the user just authorized
		var existingUserId uuid.UUID
		err = database.Pool().QueryRow(
			c,
			`
				select userid
				from auth.user_spotify
				where spotify_id=$1	
			`,
			spotifyProfile.SpotifyId,
		).Scan(&existingUserId)

		// error
		if err != nil && err != pgx.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		// no existing musicdash acc. with same linked spotify acc.
		if err == pgx.ErrNoRows {
			newUsername := spotifyProfile.DisplayName

			usernameExists, err := database.UsernameIsRegistered(c, newUsername)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
				return
			}

			emailExists, err := database.EmailIsRegistered(c, spotifyProfile.Email)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
				return
			}

			if emailExists {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "ERROR_SPOTIFY_EMAIL_REGISTERED"})
				return
			}

			if usernameExists {
				newUsername += "__musicdash_"
			}

			// create new musicdash account
			newUserId, err := database.UserInsert(newUsername, "", spotifyProfile.Email)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
				return
			}

			// obtain db.User of newly registered user

			newUser, err := database.GetUserFromId(c, newUserId)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
				return
			}

			newUser.Spotify = userSpotifyClient // save authenticated spotify client to user

			// link new account to spotify account
			if err = newUser.LinkSpotifyProfile(c, spotifyProfile); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
				return
			}

			// save spotify auth parameters to database as the user is now logged in and has a connected spotify account
			if err = newUser.SaveSpotifyAuthParams(c); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
				return
			}

			// log user into newly created account
			eventualAuthToken, err = database.UserNewAuthToken(c, newUserId)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
				return
			}
		} else {
			// existing account found, simply log into it
			eventualAuthToken, err = database.UserNewAuthToken(c, existingUserId)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ERROR_SERVER_INTERNAL"})
				return
			}
		}

		// save token as cookie on client
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", string(eventualAuthToken), int((time.Hour * 24 * 30).Seconds()), "/", "", true, true)
		c.JSON(http.StatusOK, gin.H{"token": eventualAuthToken})
	}
}

func HandlerUploadProfileImage(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser := GetUserFromCtx(c)

		// Content-Type is the first line of defense and should no tbe trusted, however
		// we require the client to set it adequately to one of the following types
		acceptedMimeTypes := []string{"image/png", "image/jpg", "image/jpeg", "image/webp"}

		contentType := c.ContentType()

		if !slices.Contains(acceptedMimeTypes, contentType) {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		contentLengthHeader := c.GetHeader("Content-Length")
		if contentLengthHeader == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		bodyLen, err := strconv.Atoi(contentLengthHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		if bodyLen > (5 * 10e6) {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		imageData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		// create new *bimg.Image from raw request image data
		image := bimg.NewImage(imageData)
		metadata, err := image.Metadata()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		var imageDataFinal []byte
		if image.Type() == "webp" {
			imageDataFinal = imageData
		} else {
			log.Println("image data provided not in web format, converting...")
			imageDataFinal, err = image.Convert(bimg.WEBP)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
				return
			}
		}

		err = currentUser.SetProfileImage(
			c,
			metadata.Size.Width,
			metadata.Size.Height,
			imageDataFinal,
		)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responseInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Profile image updated successfully."})
	}
}

// Return the profile picture of the requested user by userId.
func HandlerGetUserProfileImage(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIdParam := c.Param("userid")
		if userIdParam == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		userId, err := uuid.Parse(userIdParam)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		profileImage, err := database.GetUserProfileImage(
			c,
			userId,
		)

		if err != nil {
			if err == db.ErrNoProfileImageSet {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "ERROR_USER_NO_PROFILE_IMAGE"})
				return
			}

			c.AbortWithStatusJSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		// images are always stored in webp format
		c.Data(http.StatusOK, "image/webp", profileImage.Data)
	}
}
