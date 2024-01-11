package db

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// Check if the specified username already exists in the database.
func (db *Db) UsernameIsRegistered(username string) (bool, error) {
	row := db.pool.QueryRow(context.TODO(), "select username from auth.user where username=$1 limit 1", username)

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

	log.Printf("UUIDv4 for user {%s} = %s\n", username, userUuid.String())

	pwdAsBytes := []byte(password)

	// The password to be hashed is salted by prepending to it the unique user id (uuidv4)
	pwdToHash := make([]byte, 0, len(userUuid)+len(pwdAsBytes))
	pwdToHash = append(pwdToHash, userUuid[:]...)
	pwdToHash = append(pwdToHash, pwdAsBytes...)

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
