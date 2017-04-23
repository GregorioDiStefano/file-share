package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

var log = logrus.New()

func init() {
	log.Level = logrus.DebugLevel
}

func main() {
	config, err := readConfigFile()

	if err != nil {
		panic(err)
	}

	cloudStorage, err := newGoogleCloudStorage(
		config.googleCloudProjectName,
		config.googleBucketName,
		config.googleClientEmailID)

	if err != nil {
		log.Fatal(err.Error())
	}

	sqlConnection, err := NewSQL(
		config.mysqlUsername,
		config.mysqlPassword,
		config.mysqlHost)

	if err != nil {
		log.Fatal(err.Error())
	}

	go func() {
		for {
			time.Sleep(60 * 24)
		}

	}()

	setupRoutes(sqlConnection, cloudStorage, config)
}

func setupRoutes(db *DB, gcs *cloudStorageConfig, c *config) {
	g := gin.Default()

	g.POST("/", func(context *gin.Context) {
		id, deleteID := upload(db, gcs, context)
		context.JSON(http.StatusCreated, map[string]string{"id": id, "delete_id": deleteID})
	})

	g.GET("/:id", func(context *gin.Context) {
		id := context.Param("id")
		ip := context.ClientIP()
		fd, err := db.getFile(id)

		if err != nil {
			context.JSON(http.StatusNotFound, "unable to get specificed file")
			return
		}

		if fd.deleted {
			context.JSON(http.StatusForbidden, "this file has been deleted")
			return
		}

		if fd.downloads > c.appMaxDownloadsBeforeCaptcha {
			context.JSON(http.StatusForbidden, "this file has been downloaded too many times")
			return
		}

		if t, err := time.Parse("2006-01-02 15:04:05", fd.uploaded); err == nil {
			if t.Unix()+int64(c.fileTTL) < time.Now().UTC().Unix() {
				context.JSON(http.StatusForbidden, "this file has expired")
				return
			}
		} else {
			log.Warn("unable to verify upload datetime: %s", err)
		}

		url, err := gcs.getSignedURL(fd.key, fd.filename)

		if err != nil {
			context.JSON(http.StatusInternalServerError, "unable to generated signed url to download")
			return
		}

		go db.incDownloadCount(id) // no need to wait
		go db.addDownloadEntry(id, ip)

		context.Redirect(http.StatusTemporaryRedirect, url)
	})

	g.DELETE("/:id/:delete_id", func(context *gin.Context) {
		id := context.Param("id")
		deleteID := context.Param("delete_id")

		if _, err := db.getFile(id); err != nil {
			context.JSON(http.StatusNotFound, "unable to find requested id")
			return
		}

		if err := db.deleteFile(id, deleteID); err != nil {
			context.JSON(http.StatusUnauthorized, "failed to delete file")
			return
		}

		context.Status(http.StatusAccepted)
	})

	g.Run(":8081")
}
