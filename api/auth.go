package api

import (
	"golang.org/x/crypto/bcrypt"
	"janio-backend/helper"
	"net/http"
	"time"

	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"janio-backend/db"
	"janio-backend/model"
)


func CreateAdmin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	user := model.User{Username:username}
	// Generates a hashed version of our password
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ",err)
	}
	user.Password = string(hashedPass)
	user.IsAdmin = true
	user.DriverLicense = "admin"

	db.DbManager().Create(&user)



	return c.JSON(http.StatusCreated, echo.Map{
		"message": "Created",
	})
}


func Login(c echo.Context) error {

	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return err
	}
	username := fmt.Sprintf("%v", m["username"])
	password := fmt.Sprintf("%v", m["password"])

	user := model.User{}
	db.DbManager().Where("username = ?", username).First(&user)

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "Wrong email or password",
		})
	}


	// Set custom claims
	claims := &model.JwtCustomClaims{
		user,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 365).Unix(),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return err
	}
	user.Password=""
	user.Avatar = helper.GetS3Url(user.Avatar,"avatars")
	return c.JSON(http.StatusOK, echo.Map{
		"token": t,
		"user":&user,
	})
}


func Restricted(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)
	email := claims.User.Email
	return c.String(http.StatusOK, "Welcome "+email+"!")
}

