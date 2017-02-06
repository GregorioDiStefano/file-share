package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/scrypt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

type DB struct {
	connection *sql.DB
}

type FileData struct {
	key      string
	filename string
	filesize int64

	deleted bool

	downloads int
}

const (
	downloadKeySize = 8
	deleteKeySize   = 32

	errorIDNotFound        = "error getting specified file id"
	errorDeleteFailed      = "error deleting specified file"
	errorPasswordIncorrect = "password incorrect"
	errorNonExistingUser   = "user not found"
)

func NewSQL(user, password, ip string) (*DB, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/files?charset=utf8", user, password, ip))

	if err != nil {
		log.Warnln("failed to connect mysql server")
		return nil, err
	}

	return &DB{connection: db}, nil
}

func (db *DB) addFile(filename string, filesize int64, uploadIP string) (id, delete_id string) {
	stmt, err := db.connection.Prepare("INSERT Files SET id=?,delete_id=?,filename=?,filesize=?,upload_ip=?")

	if err != nil {
		panic(err)
	}

	id = randomString(downloadKeySize)
	delete_id = randomString(deleteKeySize)

	_, err = stmt.Exec(id, delete_id, filename, filesize, uploadIP)

	if err != nil {
		panic(err)
	}

	return
}

func (db *DB) getFile(id string) (fdata *FileData, err error) {
	var filename string
	var downloads int
	var deleted bool

	row := db.connection.QueryRow("SELECT filename, downloads, deleted FROM Files WHERE id = ?;", id)
	err = row.Scan(&filename, &downloads, &deleted)

	if err != nil {
		log.WithField("id", id).Warn("not able to load id: ", err)
		return nil, fmt.Errorf(errorIDNotFound)
	}

	log.WithFields(logrus.Fields{"key": id, "filename": filename, "downloads": downloads, "deleted": deleted}).Debug("file exists")
	return &FileData{filename: filename, key: id, downloads: downloads, deleted: deleted}, nil
}

func (db *DB) incDownloadCount(id string) (err error) {
	_, err = db.connection.Exec("UPDATE Files SET downloads = downloads + 1 WHERE id=?", id)
	return
}

func (db *DB) deleteFile(id, delete_id string) (err error) {
	stmt, err := db.connection.Prepare("UPDATE Files SET deleted=true WHERE id=? AND delete_id=?;")

	if err != nil {
		return
	}

	row, err := stmt.Exec(id, delete_id)

	if count, err := row.RowsAffected(); count == 0 || err != nil {
		return fmt.Errorf(errorDeleteFailed)
	}

	return
}

func (db *DB) createUser(signupData *IncomingSignupRequest) (err error) {
	salt := make([]byte, 16)
	rand.Read(salt)

	hash, err := scrypt.Key([]byte(signupData.Password), salt, 16384, 256, 2, 32)
	if err != nil {
		log.Warn("failed generating key for user: ", err.Error())
		return
	}

	hashEncoded := base64.RawStdEncoding.EncodeToString(hash)
	saltEncoded := base64.RawStdEncoding.EncodeToString(salt)
	stmt, err := db.connection.Prepare("INSERT Users SET username=?, password_hash=?, email=?, created=?, salt=?")

	if err != nil {
		log.Warn("failed preparing sql statement to create user: ", err.Error())
		return
	}

	_, err = stmt.Exec(signupData.Username, hashEncoded, signupData.Email, time.Now(), saltEncoded)

	if err != nil {
		log.Warn("failed executing user creation statement: ", err.Error())
		return
	}

	return
}

func (db *DB) loginUser(loginData *IncomingLoginRequest) (err error) {
	var password_hash, salt string
	row := db.connection.QueryRow("SELECT password_hash, salt FROM Users WHERE username = ?;", loginData.Username)
	err = row.Scan(&password_hash, &salt)

	if err != nil {
		switch {
		case err == sql.ErrNoRows:
			log.Warn("Non-existing user.")
			return fmt.Errorf(errorNonExistingUser)
		}
		return
	}

	passwordHashDecoded, _ := base64.RawStdEncoding.DecodeString(password_hash)
	saltDecoded, _ := base64.RawStdEncoding.DecodeString(salt)

	actualHash, err := scrypt.Key([]byte(loginData.Password), saltDecoded, 16384, 256, 2, 32)

	if err != nil {
		log.Warn("failed generating key for user: ", err.Error())
		return
	}

	if !bytes.Equal(actualHash, passwordHashDecoded) {
		return fmt.Errorf(errorPasswordIncorrect)
	}

	return
}
