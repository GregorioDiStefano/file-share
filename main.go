package main

import (
	"net/http"
	"time"

	jwt_lib "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/contrib/jwt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

	setupRoutes(sqlConnection, cloudStorage, config)
}

func setupRoutes(db *DB, gcs *cloudStorageConfig, c *config) {
	g := gin.Default()

	g.POST("/", func(context *gin.Context) {
		id, deleteID := upload(db, gcs, context)
		context.JSON(http.StatusCreated, map[string]string{"id": id, "delete_id": deleteID})
	})

	g.POST("/account/signup", func(context *gin.Context) {
		registrationReq := new(IncomingSignupRequest)
		context.BindJSON(&registrationReq)

		if err := registrationReq.validate(); err != nil {
			context.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		db.createUser(registrationReq)
	})

	private := g.Group("/user")
	private.Use(jwt.Auth(c.appSecret)).GET("/files", func(context *gin.Context) {
		context.JSON(200, "abc")
	})

	g.POST("/account/login", func(context *gin.Context) {
		loginReq := new(IncomingLoginRequest)
		context.BindJSON(&loginReq)

		if err := db.loginUser(loginReq); err != nil {
			context.JSON(http.StatusNetworkAuthenticationRequired, "unable to authenticate")
			return
		}

		token := jwt_lib.New(jwt_lib.GetSigningMethod("HS256"))

		token.Claims = jwt_lib.MapClaims{
			"id":  loginReq.Username,
			"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
		}

		tokenString, err := token.SignedString([]byte(c.appSecret))
		if err != nil {
			context.JSON(500, gin.H{"message": "Could not generate token"})
		}

		context.SetCookie("jwt", tokenString, 60, "*", "localhost", true, true)
	})

	g.GET("/x/:id", func(context *gin.Context) {
		id := context.Param("id")
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

		url, err := gcs.getSignedURL(fd.key, fd.filename)

		if err != nil {
			context.JSON(http.StatusInternalServerError, "unable to generated signed url to download")
			return
		}

		go db.incDownloadCount(id) // no need to wait
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
