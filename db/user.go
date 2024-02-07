package db

import (
	music "bool3max/musicdash/music"
	"bool3max/musicdash/spotify"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// An authentication token is generated upon successful login and consists of 64 random bytes of data,
// encoded in base64 format and stored as a string in the database.
type UserAuthToken string

var ErrEmailNotRegistered = errors.New("e-mail does not exist in database")
var ErrPasswordIncorrect = errors.New("password incorrect")
var ErrInvalidAuthToken = errors.New("invalid auth token")
var ErrSpotifyProfileNotLinked = errors.New("user has no linked spotify profile")
var ErrUserNotFound = errors.New("user not found")
var ErrNoProfileImageSet = errors.New("user has no profile image set")

type User struct {
	Id           uuid.UUID
	RegisteredAt time.Time
	Username     string
	Email        string
	Spotify      *spotify.Client
}

type UserProfileImage struct {
	Width, height int
	Data          []byte
	Size          int
	UploadedAt    time.Time
}

// Set the user's profile image. All profile images must first be converted to webp format and sanitized.
// This method performs no such checks, I do them in the http handler.
// It sets the "size" column based on the length of the provided binary data of the image.
func (user *User) SetProfileImage(ctx context.Context, width, height int, data []byte) error {
	_, err := Acquire().pool.Exec(
		ctx,
		`
			insert into auth.user_profile_img
			(userid, width, height, data, size)
			values
			($1, $2, $3, $4, $5)
			on conflict on constraint user_profile_img_pk do update
			set width=$2, height=$3, data=$4, size=$5
		`,
		user.Id,
		width,
		height,
		data,
		len(data),
	)

	return err
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

// Associate/link a Spotify user profile with a musicdash account. This should only be done once the user
// properly authenticates with Spotify. The user cannot have more than one linked Spotify account per
// musicdash account.
func (user *User) LinkSpotifyProfile(ctx context.Context, spotifyProfile spotify.UserProfile) error {
	_, err := Acquire().pool.Exec(
		ctx,
		`
			insert into auth.user_spotify
			(userid, spotify_displayname, spotify_followers, spotify_uri, profile_image_url, profile_image_width, profile_image_height, country, spotify_email, spotify_url, spotify_id)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			on conflict on constraint user_spotify_pk do update
			set spotify_displayname=$2, spotify_followers=$3, spotify_uri=$4, profile_image_url=$5, profile_image_width=$6, profile_image_height=$7, country=$8, spotify_email=$9, spotify_url=$10, spotify_id=$11
		`,
		user.Id,
		spotifyProfile.DisplayName,
		spotifyProfile.FollowerCount,
		spotifyProfile.ProfileUri,
		spotifyProfile.ProfileImages[len(spotifyProfile.ProfileImages)-1].Url,
		spotifyProfile.ProfileImages[len(spotifyProfile.ProfileImages)-1].Width,
		spotifyProfile.ProfileImages[len(spotifyProfile.ProfileImages)-1].Height,
		spotifyProfile.Country,
		spotifyProfile.Email,
		spotifyProfile.ProfileUrl,
		spotifyProfile.SpotifyId,
	)

	return err
}

func (user *User) UnlinkSpotifyProfile(ctx context.Context) error {
	_, err := Acquire().pool.Exec(
		ctx,
		`
			delete from auth.user_spotify
			where userid=$1
		`,
		user.Id,
	)

	return err
}

func (user *User) GetLinkedSpotifyProfile(ctx context.Context) (spotify.UserProfile, error) {
	var profile spotify.UserProfile
	profile.ProfileImages = make([]music.Image, 1)

	err := Acquire().pool.QueryRow(
		ctx,
		`
			select spotify_id, spotify_displayname, spotify_followers, spotify_uri, spotify_url, profile_image_url, profile_image_width, profile_image_height, country, spotify_email
			from auth.user_spotify
			where userid=$1
		`,
		user.Id,
	).Scan(&profile.SpotifyId, &profile.DisplayName, &profile.FollowerCount, &profile.ProfileUri, &profile.ProfileUrl, &profile.ProfileImages[0].Url, &profile.ProfileImages[0].Width, &profile.ProfileImages[0].Height, &profile.Country, &profile.Email)

	if err != nil {
		if err == pgx.ErrNoRows {
			return profile, ErrSpotifyProfileNotLinked
		}

		return profile, err
	}

	return profile, nil
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
	var userId uuid.UUID
	err := db.pool.QueryRow(
		ctx,
		`
			select userid
			from auth.auth_token
			where token=$1
			limit 1
		`,
		string(token),
	).Scan(&userId)

	if err != nil {
		if err == pgx.ErrNoRows {
			// auth token not in database
			return User{}, ErrInvalidAuthToken
		}

		// other db error
		return User{}, err
	}

	return db.GetUserFromId(ctx, userId)
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

// Check if there is a registered account with the specified email already in the database
func (db *Db) EmailIsRegistered(ctx context.Context, email string) (bool, error) {
	err := db.pool.QueryRow(ctx, "select email from auth.user where email=$1 limit 1", email).Scan(nil)

	switch err {
	case pgx.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}

func (db *Db) GetUserFromId(ctx context.Context, userId uuid.UUID) (User, error) {
	newUser := User{Id: userId}

	err := db.pool.QueryRow(
		ctx,
		`
			select username, email, registered_at
			from auth.user
			where id=$1
		`,
		userId,
	).Scan(&newUser.Username, &newUser.Email, &newUser.RegisteredAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, ErrUserNotFound
		}

		return User{}, err
	}

	return newUser, nil
}

// Get a user based on an unique username.
func (db *Db) GetUserFromUsername(ctx context.Context, username string) (User, error) {
	var userId uuid.UUID
	err := db.pool.QueryRow(
		ctx,
		`
			select id
			from auth.user
			where username=$1
		`,
		username,
	).Scan(&userId)

	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, ErrUserNotFound
		}

		return User{}, err
	}

	return db.GetUserFromId(ctx, userId)
}

// Insert a new user into the database. Returns the user id of the new user.
func (db *Db) UserInsert(username, password, email string) (uuid.UUID, error) {
	// generate new uuid-v4 identifier for user
	userUuid, err := uuid.NewRandom()
	if err != nil {
		return uuid.UUID{}, err
	}

	// the final password hash to be insert into database
	var pwdHash []byte = nil

	// only generate password hash if plaintext password was provided
	// otherwise, the user account is created without a password
	// (this happens when "continue with spotify" is used and the account is considered simply
	// a vessel for logging into with spotify)
	if password != "" {
		pwdAsBytes := []byte(password)

		// The password to be hashed is salted by prepending to it the unique user id (uuidv4)
		pwdToHash := make([]byte, 0, len(userUuid)+len(pwdAsBytes))
		pwdToHash = append(pwdToHash, userUuid[:]...)
		pwdToHash = append(pwdToHash, pwdAsBytes...)

		pwdHash, err = bcrypt.GenerateFromPassword(pwdToHash, bcrypt.DefaultCost)
		if err != nil {
			return uuid.UUID{}, err
		}
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
		return uuid.UUID{}, err
	}

	return userUuid, nil
}

// Attemp to validate login credentials for a user. If the provided email does not match any registered
// user in the database, an ErrEmailNotRegistered error is returned. If the provided email is registered
// but its associated password is guessed incorrectly, an ErrPasswordIncorrect error is returned.
// Otherwise, if both the email exists and the psasword is correct, error is nil.
func (db *Db) UserValidateLoginCred(ctx context.Context, passwordGuess, email string) (uuid.UUID, error) {
	passwordGuessBytes := []byte(passwordGuess)

	row := db.pool.QueryRow(
		ctx,
		`
			select id, pwdhash
			from auth.user
			where email=$1
			limit 1
		`,
		email,
	)

	var userId uuid.UUID
	pwdHashDb := make([]byte, 0)

	err := row.Scan(
		&userId,
		&pwdHashDb,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return uuid.UUID{}, ErrEmailNotRegistered
		} else {
			return uuid.UUID{}, err
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
		return uuid.UUID{}, ErrPasswordIncorrect
	}

	// login credentials correct
	return userId, nil

}

// Unconditionally issue an auth token for an user. The function generates a new valid UserAuthToken,
// saves it in the // database, and returns it. The function doesn't check if the passed userId is valid, and attempts
// to insert it into auth.auth_token, which will of course fail on an invalid user id due to the
// foreign key constraint.
func (db *Db) UserNewAuthToken(ctx context.Context, userId uuid.UUID) (UserAuthToken, error) {
	// generate new random 64 bytes to use as auth token
	authToken := make([]byte, 64)
	_, err := rand.Read(authToken)
	if err != nil {
		return "", err
	}

	// encode as base64 string
	authTokenB64 := base64.StdEncoding.EncodeToString(authToken)

	// insert the generated auth token into the database

	// preserve token into database
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

// Get the current set profile image of the specified user. If the user has no profile image set,
// an ErrNoProfileImageSet error is returned and a default profile image should be supplied.
func (db *Db) GetUserProfileImage(ctx context.Context, userId uuid.UUID) (UserProfileImage, error) {
	newProfileImg := UserProfileImage{}
	err := db.pool.QueryRow(
		ctx,
		`
			select width, height, size, data, uploaded_at
			from auth.user_profile_img
			where userid=$1
		`,
		userId,
	).Scan(&newProfileImg.Width, &newProfileImg.height, &newProfileImg.Size, &newProfileImg.Data, &newProfileImg.UploadedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return UserProfileImage{}, ErrNoProfileImageSet
		}

		return UserProfileImage{}, err
	}

	return newProfileImg, nil
}
