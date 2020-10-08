package api

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"janio-backend/db"
	"janio-backend/helper"
	"janio-backend/model"
	"janio-backend/permission"
	"janio-backend/services"
	"janio-backend/validator"
	"net/http"
	"strconv"
)

func UpdateDriver(c echo.Context) error {
	userAuth := c.Get("user").(*jwt.Token)
	claims := userAuth.Claims.(*model.JwtCustomClaims)

	if permission.AdminPermission(claims) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}
	//find
	driverId := c.Param("id")
	user := model.User{}
	db.DbManager().Where("id = ?", driverId).Find(&user)
	if user.ID == 0 {
		payload := &PayloadError{
			Errors: "Driver not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}


	//get param and set
	DriverLicense := c.FormValue("driver_license")
	LicensePlate := c.FormValue("license_plate")
	DriverName := c.FormValue("driver_name")
	Phone := c.FormValue("phone")
	PostalCode := c.FormValue("postal_code")
	ShiftPickupStart := c.FormValue("shift_pickup_start")
	ShiftPickupEnd := c.FormValue("shift_pickup_end")
	ShiftDeliveryStart := c.FormValue("shift_delivery_start")
	ShiftDeliveryEnd := c.FormValue("shift_delivery_end")
	DriverType := c.FormValue("driver_type")
	Weight := c.FormValue("weight")
	Volume := c.FormValue("volume")
	Country := c.FormValue("country")
	Timezone := c.FormValue("timezone")

	//check each postal not belong more than one driver


	user.DriverLicense = DriverLicense
	user.LicensePlate = LicensePlate
	user.DriverName = DriverName
	user.Phone = Phone
	user.PostalCode = PostalCode
	user.ShiftPickupStart = ShiftPickupStart
	user.ShiftPickupEnd = ShiftPickupEnd
	user.ShiftDeliveryStart = ShiftDeliveryStart
	user.ShiftDeliveryEnd = ShiftDeliveryEnd
	user.DriverType = DriverType
	user.Weight = Weight
	user.Volume = Volume
	user.Country = Country
	user.Timezone = Timezone

	//add validator
	errs :=  validator.UpdateDriverValidator(&user)
	if len(errs) > 0 {
		payload := &PayloadError{
			Errors: errs,
		}
		return c.JSON(http.StatusBadRequest, payload)
	}


	if user.Timezone == "" {
		user.Timezone = "Asia/Singapore"
	}
	if user.Country == "" {
		user.Country = "Singapore"
	}

	// Update avatar
	fileStream, err := c.FormFile("avatar")

	if fileStream != nil {
		//VALIDATE FILE
		if services.SizeAllow(fileStream, maxUploadSize) == false {
			payload := &PayloadError{
				Errors: "File size must " + string(maxUploadSize) + "mb or below",
			}
			return c.JSON(http.StatusBadRequest, payload)
		}
		if services.IsImage(fileStream) == false {
			payload := &PayloadError{
				Errors: "file should be an image",
			}
			return c.JSON(http.StatusBadRequest, payload)
		}
		//END VALIDATE FILE
		fileName := services.UploadS3(fileStream, "avatars")
		user.Avatar = fileName
	}

	// Generates a hashed version of our password
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(user.DriverLicense), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}
	user.Password = string(hashedPass)

	db.DbManager().Save(&user)

	user.Avatar = helper.GetS3Url(user.Avatar, "avatars")
	payload := &PayloadSuccess{
		Data: &user,
	}
	return c.JSON(http.StatusOK, payload)
}

func ListDriver(c echo.Context) error {
	userAuth := c.Get("user").(*jwt.Token)
	claims := userAuth.Claims.(*model.JwtCustomClaims)

	if permission.AdminPermission(claims) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}


	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	driverLicense := c.QueryParam("driver_license")
	licensePlate := c.QueryParam("license_plate")

	// Defaults
	if offset == 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 100000
	}

	db := db.DbManager()
	users := []*model.User{}
	count := 0
	query := db.Offset(offset).Limit(limit).Where("is_admin = 0")
	queryCount := db.Table("users").Where("is_admin = 0")
	if driverLicense != "" {
		query = query.Where("driver_license LIKE ?", "%"+driverLicense+"%")
		queryCount = queryCount.Where("driver_license LIKE ?", "%"+driverLicense+"%")
	}
	if licensePlate != "" {
		query = query.Where("license_plate LIKE ?", "%"+licensePlate+"%")
		queryCount = queryCount.Where("license_plate LIKE ?", "%"+licensePlate+"%")
	}
	query.Find(&users)
	queryCount.Count(&count)

	for _, user := range users {
		user.Avatar = helper.GetS3Url(user.Avatar, "avatars")
	}

	payload := &PayloadSuccess{
		Data: users,
		Meta: struct {
			TotalRecord int `json:"total_record"`
		}{count},
	}
	return c.JSON(http.StatusOK, payload)
}

func CreateDriver(c echo.Context) (err error) {
	userAuth := c.Get("user").(*jwt.Token)
	claims := userAuth.Claims.(*model.JwtCustomClaims)

	if permission.AdminPermission(claims) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}
	//db := db.DbManager()

	//get param and set
	user := &model.User{}

	DriverLicense := c.FormValue("driver_license")
	LicensePlate := c.FormValue("license_plate")
	DriverName := c.FormValue("driver_name")
	Phone := c.FormValue("phone")
	PostalCode := c.FormValue("postal_code")
	ShiftPickupStart := c.FormValue("shift_pickup_start")
	ShiftPickupEnd := c.FormValue("shift_pickup_end")
	ShiftDeliveryStart := c.FormValue("shift_delivery_start")
	ShiftDeliveryEnd := c.FormValue("shift_delivery_end")
	DriverType := c.FormValue("driver_type")
	Weight := c.FormValue("weight")
	Volume := c.FormValue("volume")
	Country := c.FormValue("country")
	Timezone := c.FormValue("timezone")

	//check each postal not belong more than one driver


	user.DriverLicense = DriverLicense
	user.LicensePlate = LicensePlate
	user.DriverName = DriverName
	user.Phone = Phone
	user.PostalCode = PostalCode
	user.ShiftPickupStart = ShiftPickupStart
	user.ShiftPickupEnd = ShiftPickupEnd
	user.ShiftDeliveryStart = ShiftDeliveryStart
	user.ShiftDeliveryEnd = ShiftDeliveryEnd
	user.DriverType = DriverType
	user.Weight = Weight
	user.Volume = Volume
	user.Country = Country
	user.Timezone = Timezone
	//add validator
	errs :=  validator.CreateDriverValidator(user)
	if len(errs) > 0 {
		payload := &PayloadError{
			Errors: errs,
		}
		return c.JSON(http.StatusBadRequest, payload)
	}

	// Update avatar
	fileStream, err := c.FormFile("avatar")

	if fileStream != nil {
		fileName := services.UploadS3(fileStream, "avatars")
		user.Avatar = fileName
	}
	//save ro db
	// Generates a hashed version of our password
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(user.DriverLicense), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}
	user.Password = string(hashedPass)
	user.Username = user.DriverLicense
	user.IsAdmin = false



	if user.Timezone == "" {
		user.Timezone = "Asia/Singapore"
	}
	if user.Country == "" {
		user.Country = "Singapore"
	}

	db.DbManager().Create(&user)

	if user.Avatar != "" {
		user.Avatar = helper.GetS3Url(user.Avatar, "avatars")
	}
	user.Password = "" // Don't send password
	return c.JSON(http.StatusCreated, user)
}
