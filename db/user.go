package db

import (
	"bool3max/musicdash/spotify"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// An authentication token is generated upon successful login and consists of 64 random bytes of data,
// encoded in base64 format and stored as a string in the database.
type UserAuthToken string

var ErrorEmailNotRegistered = errors.New("e-mail does not exist in database")
var ErrorPasswordIncorrect = errors.New("password incorrect")
var ErrorInvalidAuthToken = errors.New("invalid auth token")

type User struct {
	Id       uuid.UUID
	Username string
	Email    string
	Spotify  *spotify.Client
}

// Preserve the current parameters in user.Spotify to the database unconditionally.
func (user *User) SaveSpotifyAuthParams(ctx context.Context) error {
	if user.Spotify == nil {
		return nil
	}

	_, err := Acquire().pool.Exec(
		ctx,
		`
			insert into auth.spotify_token
			(userid, accesstoken, refreshtoken, expiresat)
			values ($1, $2, $3, $4)
			on conflict on constraint spotify_token_pk do update
			set accesstoken=$2, refreshtoken=$3, expiresat=$4
		`,
		user.Id,
		user.Spotify.AccessToken,
		user.Spotify.RefreshToken,
		user.Spotify.ExpiresAt,
	)

	return err
}

func (user *User) String() string {
	return fmt.Sprintf("[id:{%s}, username:{%s}, email:{%s}]", user.Id.String(), user.Username, user.Email)
}

func (user *User) RevokeAllTokens(ctx context.Context) error {
	_, err := Acquire().pool.Exec(
		ctx,
		`
			delete from auth.auth_token
			where auth.auth_token.userid=$1
		`,
		user.Id,
	)

	return err
}

// Validate the passed UserAuthToken and return an instance of the User that it belongs to.
// If the passed auth. token is invalid (i.e. does not exist in the database), return an
// ErrorInvalidAuthToken error and an empty User{} instance.
func (db *Db) GetUserFromAuthToken(ctx context.Context, token UserAuthToken) (User, error) {
	newUser := User{}
	err := db.pool.QueryRow(
		ctx,
		`
			select userid
			from auth.auth_token
			where token=$1
			limit 1
		`,
		string(token),
	).Scan(&newUser.Id)

	if err != nil {
		if err == pgx.ErrNoRows {
			// auth token not in database
			return newUser, ErrorInvalidAuthToken
		} else {
			// other db error
			return newUser, err
		}
	}

	err = db.pool.QueryRow(
		ctx,
		`
			select username, email
			from auth.user
			where id=$1
		`,
		newUser.Id,
	).Scan(&newUser.Username, &newUser.Email)

	if err != nil {
		return User{}, err
	}

	return newUser, nil
}

// Check if the specified username already exists in the database.
func (db *Db) UsernameIsRegistered(ctx context.Context, username string) (bool, error) {
	row := db.pool.QueryRow(ctx, "select username from auth.user where username=$1 limit 1", username)

	err := row.Scan(nil)

	switch err {
	case pgx.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}

func (db *Db) UserInsert(username, password, email string) error {
	// generate new uuid-v4 identifier for user
	userUuid, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	log.Printf("signup: new UUIDv4 for user {%s} = %s\n", username, userUuid.String())

	pwdAsBytes := []byte(password)

	// The password to be hashed is salted by prepending to it the unique user id (uuidv4)
	pwdToHash := make([]byte, 0, len(userUuid)+len(pwdAsBytes))
	pwdToHash = append(pwdToHash, userUuid[:]...)
	pwdToHash = append(pwdToHash, pwdAsBytes...)

	log.Printf("signup: pwdToHash len: %v\n", len(pwdToHash))

	pwdHash, err := bcrypt.GenerateFromPassword(pwdToHash, bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	sqlQueryInsertNewUser := `
		insert into auth.user
		(id, username, pwdhash, email)	
		values ($1, $2, $3, $4)
	`

	_, err = db.pool.Exec(
		context.TODO(),
		sqlQueryInsertNewUser,
		userUuid,
		username,
		pwdHash,
		email,
	)

	if err != nil {
		return err
	}

	return nil
}

// Attempt to login the user based on an email and password. If the credentials match,
// that is, if an user account with the specified e-mail is registed and the password is correct,
// an AuthToken is returned and error is nil. If the e-mail is not registered or if the password is incorrect,
// an empty AuthToken is returned (empty string), and error is set accordingly.
// After a successful login, the generated AuthToken is saved in the database and facilitates a valid
// session for that particular user account. Valid AuthToken(s) in the database have no expiration date
// and last indefinitely - that is until the user logs out.
func (db *Db) UserLogin(ctx context.Context, passwordGuess, email string) (UserAuthToken, error) {
	passwordGuessBytes := []byte(passwordGuess)

	row := db.pool.QueryRow(
		ctx,
		`
			select id, username, pwdhash
			from auth.user
			where email=$1
			limit 1
		`,
		email,
	)

	var userId uuid.UUID
	var username string
	pwdHashDb := make([]byte, 0)

	err := row.Scan(
		&userId,
		&username,
		&pwdHashDb,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrorEmailNotRegistered
		} else {
			return "", err
		}
	}

	// A properly salted byte slice of the password the user guessed, as a []byte
	passwordGuessSaltedBytes := make([]byte, 0, len(userId)+len(passwordGuessBytes))
	passwordGuessSaltedBytes = append(passwordGuessSaltedBytes, userId[:]...)
	passwordGuessSaltedBytes = append(passwordGuessSaltedBytes, passwordGuessBytes...)

	// compare correct password in db and guess
	err = bcrypt.CompareHashAndPassword(pwdHashDb, passwordGuessSaltedBytes)
	// passwords do not match
	if err != nil {
		log.Printf("error comparing passwords: %v\n", err)
		return "", ErrorPasswordIncorrect
	}

	// passwords match, generate auth token
	authToken := make([]byte, 64)
	_, err = rand.Read(authToken)
	if err != nil {
		return "", err
	}

	authTokenB64 := base64.StdEncoding.EncodeToString(authToken)

	// insert the generated auth token into the database

	_, err = db.pool.Exec(
		ctx,
		`
			insert into auth.auth_token
			(userid, token)
			values ($1, $2)
		`,
		userId,
		authTokenB64,
	)

	if err != nil {
		return "", err
	}

	return UserAuthToken(authTokenB64), nil
}

// Revoke a specific auth. token
func (db *Db) RevokeAuthToken(ctx context.Context, token UserAuthToken) error {
	_, err := db.pool.Exec(
		ctx,
		`
			delete from auth.auth_token
			where auth.auth_token.token = $1	
		`,
		token,
	)

	return err
}
