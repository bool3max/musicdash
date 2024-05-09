package db

import (
	music "bool3max/musicdash/music"
	"bool3max/musicdash/spotify"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
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

// Get the last N plays made by the corresponding user, order by most recent play first.
// An alternative Spotify ResourceProvider must be passed in to handle cases where a track
// isn't preserved in the database.
func (user *User) GetRecentPlaysFromDB(limit int, spotifyProvider music.ResourceProvider) ([]spotify.Play, error) {
	rows, err := Acquire().pool.Query(
		context.Background(),
		`
			select spotifyid, at
			from public.plays
			where userid=$1
			order by at desc
			limit $2
		`,
		user.Id,
		limit,
	)

	if err != nil {
		return nil, err
	}

	plays, err := pgx.CollectRows[spotify.Play](rows, func(row pgx.CollectableRow) (spotify.Play, error) {
		var spotifyId string
		var at time.Time

		if err := row.Scan(&spotifyId, &at); err != nil {
			return spotify.Play{}, err
		}

		// attempt to get track data from local database
		track, err := Acquire().GetTrackById(spotifyId)

		// found track preserved in database
		if err == nil {
			return spotify.Play{
				At:    at,
				Track: *track,
			}, nil
		}

		// track not in database, get it from Spotify
		if err == ErrResourceNotPreserved {
			track, err := spotifyProvider.GetTrackById(spotifyId)
			if err != nil {
				return spotify.Play{}, err
			}

			return spotify.Play{
				Track: *track,
				At:    at,
			}, nil
		}

		// other error?
		return spotify.Play{}, err
	})

	return plays, nil
}

// Return a slice of registered users who have a linked Spotify client. User.Spotify clients
// are not initialized.
func (db *Db) GetUsersWithSpotifyLinked() ([]User, error) {
	users := make([]User, 0)

	rows, err := db.pool.Query(
		context.Background(),
		`
			select id, username, registered_at, email
			from auth.user_spotify	
				inner join auth.user on auth.user.id=auth.user_spotify.userid
		`,
	)

	if err != nil {
		return nil, err
	}

	var (
		id           uuid.UUID
		username     string
		email        string
		registeredAt time.Time
	)

	pgx.ForEachRow(rows, []any{&id, &username, &registeredAt, &email}, func() error {
		users = append(users, User{
			Username:     username,
			Email:        email,
			RegisteredAt: registeredAt,
			Id:           id,
		})

		return nil
	})

	return users, nil
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
			(@userId, @width, @height, @data, @size)
			on conflict on constraint user_profile_img_pk do update
			set width=@width, height=@height, data=@data, size=@size
		`,
		pgx.NamedArgs{
			"userId": user.Id,
			"width":  width,
			"height": height,
			"data":   data,
			"size":   len(data),
		},
	)

	return err
}

// Establish a user.Spotify client using authentication parameters from the database
func (user *User) GetSpotifyAuth(ctx context.Context) error {
	var accessToken, refreshToken string
	var expiresAt time.Time

	err := Acquire().pool.QueryRow(
		ctx,
		`
				select accesstoken, refreshtoken, expiresat
				from auth.spotify_token
				where userid=$1
			`,
		user.Id,
	).Scan(&accessToken, &refreshToken, &expiresAt)

	if err != nil {
		// No spotify auth. params. in db for current user -> profile not linked
		if err == pgx.ErrNoRows {
			return ErrSpotifyProfileNotLinked
		}

		return err
	}

	userSpotifyClient := spotify.AuthorizationCodeFromParams(
		MUSICDASH_SPOTIFY_CLIENT_ID,
		MUSICDASH_SPOTIFY_SECRET,
		accessToken,
		refreshToken,
		expiresAt,
	)

	// Refresh access token
	if _, err = userSpotifyClient.Refresh(); err != nil {
		log.Printf("error refreshing Spotify token for user {%s}: %v\n", user.Id.String(), err)
		return err
	}

	user.Spotify = userSpotifyClient

	// save potentially-refreshed new spotify auth. params. to database

	if err = user.SaveSpotifyAuthDB(ctx); err != nil {
		log.Println("Error preserving Spotify AuthParams to database: ", err)
		return err
	}

	return nil
}

// Preserve the current parameters in user.Spotify to the database unconditionally.
func (user *User) SaveSpotifyAuthDB(ctx context.Context) error {
	if user.Spotify == nil {
		return nil
	}

	_, err := Acquire().pool.Exec(
		ctx,
		`
			insert into auth.spotify_token
			(userid, accesstoken, refreshtoken, expiresat)
			values (@userId, @accessToken, @refreshToken, @expiresAt)
			on conflict on constraint spotify_token_pk do update
			set accesstoken=@accessToken, refreshtoken=@refreshToken, expiresat=@expiresAt
		`,
		pgx.NamedArgs{
			"userId":       user.Id,
			"accessToken":  user.Spotify.AccessToken,
			"refreshToken": user.Spotify.RefreshToken,
			"expiresAt":    user.Spotify.ExpiresAt,
		},
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
			values (@userId, @spotifyDisplayName, @spotifyFollowers, @spotifyUri, @profileImageUrl, @profileImageWidth, @profileImageHeight, @country, @spotifyEmail, @spotifyUrl, @spotifyId)
			on conflict on constraint user_spotify_pk do update
			set spotify_displayname=@spotifyDisplayName, spotify_followers=@spotifyFollowers, spotify_uri=@spotifyUri, profile_image_url=@profileImageUrl, profile_image_width=@profileImageWidth, profile_image_height=@profileImageHeight, country=@country, spotify_email=@spotifyEmail, spotify_url=@spotifyUrl, spotify_id=@spotifyId
		`,
		pgx.NamedArgs{
			"userId":             user.Id,
			"spotifyDisplayName": spotifyProfile.DisplayName,
			"spotifyFollowers":   spotifyProfile.FollowerCount,
			"spotifyUri":         spotifyProfile.ProfileUri,
			"profileImageUrl":    spotifyProfile.ProfileImages[len(spotifyProfile.ProfileImages)-1].Url,
			"profileImageHeight": spotifyProfile.ProfileImages[len(spotifyProfile.ProfileImages)-1].Height,
			"profileImageWidth":  spotifyProfile.ProfileImages[len(spotifyProfile.ProfileImages)-1].Width,
			"country":            spotifyProfile.Country,
			"spotifyEmail":       spotifyProfile.Email,
			"spotifyUrl":         spotifyProfile.ProfileUrl,
			"spotifyId":          spotifyProfile.SpotifyId,
		},
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
	err := db.pool.QueryRow(ctx, "select username from auth.user where username=$1 limit 1", username).Scan(nil)

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
		values (@id, @username, @pwdHash, @email)
	`

	_, err = db.pool.Exec(
		context.TODO(),
		sqlQueryInsertNewUser,
		pgx.NamedArgs{
			"id":       userUuid,
			"username": username,
			"pwdHash":  pwdHash,
			"email":    email,
		},
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
			values (@userId, @authToken)
		`,
		pgx.NamedArgs{
			"userId":    userId,
			"authToken": authTokenB64,
		},
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

// Unconditionally save all plays in the "plays" slice to the database and
// associate them with the given user.
// TODO: do these inserts in a transaction!!
func (user *User) SavePlays(plays []spotify.Play) error {
	db := Acquire()

	for _, play := range plays {

		_, err := db.pool.Exec(
			context.Background(),
			`
				insert into public.plays
				(userid, at, spotifyid)
				values (@userId, @at, @spotifyId)
			`,
			pgx.NamedArgs{
				"userId":    user.Id,
				"at":        play.At,
				"spotifyId": play.Track.SpotifyId,
			},
		)

		if err != nil {
			log.Printf("SavePlays: error saving play {%s}@{%v} for {%v}: %v\n", play.Track.SpotifyId, play.At, user.Id.String(), err)
			return err
		}
	}

	return nil
}
