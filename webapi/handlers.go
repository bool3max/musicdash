package webapi

import (
	"bool3max/musicdash/db"
	"net/http"
	"net/mail"

	"github.com/gin-gonic/gin"
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

var responseServerError = gin.H{"message": "Server error."}
var responseBadRequest = gin.H{"message": "Bad request"}

// Gin handler for signing up using an email and password.
func HandlerSignupCred(database *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data SignupCredRequestData
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, responseBadRequest)
			return
		}

		if len(data.Username) < 3 || len(data.Username) > 30 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Username length must be between 3 and 30 characters."})
			return
		}

		exists, err := database.UsernameIsRegistered(c, data.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, responseServerError)
			return
		}

		if exists {
			c.JSON(http.StatusForbidden, gin.H{"message": "An account with that username already exists."})
			return
		}

		// bcrypt max password byte length is 72 bytes and we salt it with a uuidv4 which is 16 bytes
		if len(data.Password) < 8 || len(data.Password) > (72-16) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Password length must be at least 8 characters and no more than 56 characters."})
			return
		}

		_, err = mail.ParseAddress(data.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid e-mail address."})
			return
		}

		if err = database.UserInsert(data.Username, data.Password, data.Email); err != nil {
			c.JSON(http.StatusInternalServerError, responseServerError)
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

		c.JSON(http.StatusOK, gin.H{"token": authToken})
	}
}
