package main

import (
	"bytes"

	_ "github.com/GregorioDiStefano/go-file-storage/log"
	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

func upload(db *DB, gcs *cloudStorageConfig, c *gin.Context) (id, delete_id string) {
	file, header, err := c.Request.FormFile("file")

	if err != nil {
		panic(err)
	}

	filename := header.Filename

	buff := bytes.Buffer{}
	filesize, err := buff.ReadFrom(file)

	// reset file to beginning after read
	file.Seek(0, 0)

	if err != nil {
		panic(err)
	}

	id, delete_id = db.addFile(filename, filesize, c.ClientIP())

	if err := gcs.uploadFile(id, filename, file); err != nil {
		log.WithFields(logrus.Fields{"key": id, "filename": filename}).Error("error upload file")
	}

	return
}
