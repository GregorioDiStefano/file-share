package main

import (
	"math/rand"
	"time"
)

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	characters := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	returnString := ""
	for i := 0; i < length; i++ {
		returnString += string(characters[rand.Intn(len(characters))])
	}

	return returnString
}
