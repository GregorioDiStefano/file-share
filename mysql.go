package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/scrypt"

	"github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	"github.com/satori/go.uuid"
)

type DB struct {
	connection *sql.DB
}

type FileData struct {
	key       string
	filename  string
	filesize  int64
	uploaded  string
	deleted   bool
	downloads int
}

const (
	downloadKeySize = 6

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

func (db *DB) addFile(filename string, filesize int64, uploadIP string) (id, delete_key string) {
	stmt, err := db.connection.Prepare("INSERT Files SET id=?,delete_id=?,filename=?,filesize=?,upload_ip=?")

	if err != nil {
		panic(err)
	}

	id = randomString(downloadKeySize)
	delete_key = uuid.NewV4().String()

	_, err = stmt.Exec(id, delete_key, filename, filesize, uploadIP)

	if err != nil {
		panic(err)
	}

	return
}

func (db *DB) getFile(id string) (fdata *FileData, err error) {
	var filename string
	var downloads int
	var deleted bool
	var uploaded string

	row := db.connection.QueryRow("SELECT filename, downloads, deleted, uploaded FROM Files WHERE id = ?;", id)
	err = row.Scan(&filename, &downloads, &deleted, &uploaded)

	if err != nil {
		log.WithField("id", id).Warn("not able to load id: ", err)
		return nil, fmt.Errorf(errorIDNotFound)
	}

	log.WithFields(logrus.Fields{"key": id, "filename": filename, "downloads": downloads, "deleted": deleted, "uploaded": uploaded}).Debug("file exists")
	return &FileData{filename: filename, key: id, downloads: downloads, deleted: deleted, uploaded: uploaded}, nil
}

func (db *DB) incDownloadCount(id string) (err error) {
	_, err = db.connection.Exec("UPDATE Files SET downloads = downloads + 1 WHERE id=?", id)
	return
}

func (db *DB) addDownloadEntry(id string, ip string) (err error) {
	stmt, err := db.connection.Prepare("INSERT Downloads SET datetime=?,ip_address=?,file_id=?")

	if err != nil {
		panic(err)
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	res, err := stmt.Exec(timestamp, ip, id)

	if count, err := res.RowsAffected(); count == 0 || err != nil {
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
