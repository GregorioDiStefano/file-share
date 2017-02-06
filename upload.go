package main

import (
	"bytes"

	"github.com/gin-gonic/gin"

	"fmt"
)

func upload(db *DB, gcs *cloudStorageConfig, c *gin.Context) (id, delete_id string) {
	file, header, err := c.Request.FormFile("upload")

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

	fmt.Println(gcs.uploadFile(id, filename, file))

	return
}
