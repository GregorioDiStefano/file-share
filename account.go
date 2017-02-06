package main

import (
	"fmt"

	"github.com/asaskevich/govalidator"
)

type IncomingSignupRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type IncomingLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (signup *IncomingSignupRequest) validate() (err error) {

	if len(signup.Username) <= 4 || len(signup.Username) > 32 {
		return fmt.Errorf("username must be greater than 4 characters and less than 32 characters")
	} else if len(signup.Password) < 8 {
		return fmt.Errorf("password must be 8 characters or longer")
	} else if !govalidator.IsEmail(signup.Email) {
		return fmt.Errorf("email is not valid")
	}
	return
}
