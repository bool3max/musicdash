package webapi

import (
	"bool3max/musicdash/db"
	"net/http"
	"net/mail"

	"github.com/gin-gonic/gin"
)

type SignupClassicRequestData struct {
	Username string `binding:"required"`
	Email    string `binding:"required"`
	Password string `binding:"required"`
}

var responseServerError = gin.H{"message": "Server error."}

// Gin handler for signing up using an email and password.
func HandlerSignupClassic(db *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data SignupClassicRequestData
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Bad request"})
			return
		}

		if len(data.Username) < 3 || len(data.Username) > 30 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Username length must be between 3 and 30 characters."})
			return
		}

		exists, err := db.UsernameIsRegistered(c, data.Username)
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

		if err = db.UserInsert(data.Username, data.Password, data.Email); err != nil {
			c.JSON(http.StatusInternalServerError, responseServerError)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Account created successfully."})
	}
}

func HandlerLoginCred(db *db.Db) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}
