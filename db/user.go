package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
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

	log.Printf("user id from database: {%v}, len: {%v}\n", userId.String(), len(userId))

	// A properly salted byte slice of the password the user guessed, as a []byte
	passwordGuessSaltedBytes := make([]byte, 0, len(userId)+len(passwordGuessBytes))
	passwordGuessSaltedBytes = append(passwordGuessSaltedBytes, userId[:]...)
	passwordGuessSaltedBytes = append(passwordGuessSaltedBytes, passwordGuessBytes...)

	log.Printf("login: passwordGuessSaltedBytes len: %v\n", len(passwordGuessSaltedBytes))

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

func (db *Db) UserRevokeToken(ctx context.Context, token UserAuthToken) error {
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
