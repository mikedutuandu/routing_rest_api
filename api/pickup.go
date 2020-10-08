package api

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"janio-backend/db"
	"janio-backend/helper"
	"janio-backend/model"
	"janio-backend/permission"
	"janio-backend/services"
	"janio-backend/validator"
	"net/http"
	"strconv"
	"time"
)

func OptimizePickup(c echo.Context) error {
	defer c.Request().Body.Close()

	b, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]float64)

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	//add validator
	errs :=  validator.OptimizePickupValidator(d)
	if len(errs) > 0 {
		payload := &PayloadError{
			Errors: errs,
		}
		return c.JSON(http.StatusBadRequest, payload)
	}

	lat := d["lat"]
	lng := d["lng"]

	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)
	DriverId := claims.User.ID

	driverTimezone := claims.User.Timezone
	today := time.Now().Format(layoutISO)
	loc, err := time.LoadLocation(driverTimezone)
	if err == nil {
		today = time.Now().In(loc).Format(layoutISO)
	}

	//update pickups of this driver
	pickups := []*model.Pickup{}
	query := db.DbManager().Order("created_at desc")
	query = query.Where("pickup_date = ?", today)
	query = query.Where("driver_id = ? ", DriverId)
	query = query.Where("status IN (?) ", []string{"SCHEDULED", "IN_PROGRESS"})

	query.Find(&pickups)

	//update orders related to pickup
	for _, pickup := range pickups {
		//update pickup
		pickup.Status = "IN_PROGRESS"
		pickup.SolutionOrder = 9999
		db.DbManager().Save(&pickup)

		//update orders relate to pickup
		orders := []*model.Order{}
		query = db.DbManager().Where("pickup_id = ?", pickup.ID)
		query = query.Where("status IN (?)", []string{"PICK_UP_SCHEDULED", "PICK_UP_IN_PROGRESS"})
		query.Find(&orders)
		for _, order := range orders {
			order.Status = "PICK_UP_IN_PROGRESS"
			db.DbManager().Save(&order)
		}
	}
	//create job
	Job := model.OptimizeJob{}
	Job.DriverId = int(DriverId)
	Job.Status = "PENDING"
	Job.JobType = "PICKUP"
	Job.CreatedAt = time.Now()
	Job.UpdatedAt = time.Now()
	db.DbManager().Save(&Job)

	jobId := strconv.Itoa(int(Job.ID))

	//optimize
	respJobId := helper.GetPickupSolution(int(DriverId), pickups, lat, lng, jobId)
	Job.JobId = respJobId
	db.DbManager().Save(&Job)

	payload := &PayloadSuccess{
		Data: &Job,
	}
	return c.JSON(http.StatusOK, payload)
}

func UpdatePickup(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)
	isAdmin := claims.User.IsAdmin


	//find
	pickupId := c.Param("id")
	pickup := model.Pickup{}
	db.DbManager().Where("id = ?", pickupId).Find(&pickup)

	if pickup.ID == 0 {
		payload := &PayloadError{
			Errors: "Pickup not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	if permission.OwnerPermissionPickup(claims,&pickup) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}

	//get param and set
	listOrderStatus := c.FormValue("list_order_status")
	driverNote := c.FormValue("driver_note")
	rescheduleDateStr := c.FormValue("reschedule_date")
	if rescheduleDateStr != "" {
		rescheduleDate, err := time.Parse(layoutISO, rescheduleDateStr)
		if err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
		pickup.RescheduleDate = rescheduleDate
	}

	//ADD VALIDATE LIST STATUS
	if validator.ValidateCanUpdatePickup(isAdmin, pickup.Status) == false {
		payload := &PayloadError{
			Errors: "This pickup is COMPLETED, cannot change anymore",
		}
		return c.JSON(http.StatusBadRequest, payload)
	}
	pickupIdInt, _ := strconv.Atoi(pickupId)

	canPick,numberOrderPicked := validator.ValidateOrderStatusListForPickup(pickupIdInt, listOrderStatus)
	if canPick == false {

		payload := &PayloadError{
			Errors: "Invalid status! status for orders relate this pickup must one of (ORDER_PICKED_UP, PICKUP_FAILED)",
		}
		return c.JSON(http.StatusBadRequest, payload)

	}
	//END VALIDATE

	pickup.Status = helper.UpdateStatusOrdersForPickup(listOrderStatus)
	pickup.DriverNote = driverNote
	pickup.TotalNumberOrderPickup = numberOrderPicked

	// Update signature
	fileStream, err := c.FormFile("customer_signature")
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}

	orders := []*model.Order{}
	db.DbManager().Where("pickup_id = ?", pickup.ID).Find(&orders)
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
		fileName := services.UploadS3(fileStream, "order-signatures")
		pickup.CustomerSignature = fileName

		//ADD TO ORDER
		for _, order := range orders {
			order.CustomerSignatureFirstMile = fileName
			order.PickupNote = pickup.DriverNote
			db.DbManager().Save(&order)
		}

	}

	db.DbManager().Save(&pickup)

	pickup.Orders = orders

	pickup.CustomerSignature = helper.GetS3Url(pickup.CustomerSignature, "order-signatures")
	payload := &PayloadSuccess{
		Data: &pickup,
	}
	return c.JSON(http.StatusOK, payload)
}

func DetailPickup(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)

	pickupId := c.Param("id")

	pickup := model.Pickup{}

	db.DbManager().Where("id = ?", pickupId).Find(&pickup)

	if pickup.ID == 0 {
		payload := &PayloadError{
			Errors: "Pickup not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	if permission.OwnerPermissionPickup(claims,&pickup) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}


	orders := []*model.Order{}
	db.DbManager().Where("pickup_id = ?", pickupId).Find(&orders)

	pickup.Orders = orders

	pickup.CustomerSignature = helper.GetS3Url(pickup.CustomerSignature, "order-signatures")

	pods := []*model.PodPickup{}
	db.DbManager().Where("pickup_id = ?", pickupId).Find(&pods)

	for _, pod := range pods {
		pod.Image = helper.GetS3Url(pod.Image, "pods")
	}

	pickup.Pods = pods

	payload := &PayloadSuccess{
		Data: pickup,
	}

	return c.JSON(http.StatusOK, payload)
}

func ListPickup(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)
	isAdmin := claims.User.IsAdmin

	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	pickupDate := c.QueryParam("pickup_date")

	// Defaults
	if offset == 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 10
	}

	db := db.DbManager()
	pickup := []*model.Pickup{}
	count := 0
	query := db.Offset(offset).Limit(limit).Order("solution_order asc")
	queryCount := db.Table("pickups")

	if pickupDate != "" {
		query = query.Where("DATE(pickup_date) = ?", pickupDate)
		queryCount = queryCount.Where("DATE(pickup_date) = ?", pickupDate)
	}
	if isAdmin == false {
		DriverId := claims.User.ID
		if DriverId != 0 {
			query = query.Where("driver_id = ? ", DriverId)
			queryCount = queryCount.Where("driver_id  = ?", DriverId)
		}
	}

	query.Find(&pickup)
	queryCount.Count(&count)

	payload := &PayloadSuccess{
		Data: pickup,
		Meta: struct {
			TotalRecord int `json:"total_record"`
		}{count},
	}
	return c.JSON(http.StatusOK, payload)
}

func UploadPODPickup(c echo.Context) error {

	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)

	pickupId := c.Param("id")
	pickupIdInt, _ := strconv.Atoi(pickupId)

	fileStream, _ := c.FormFile("image")

	pickup := model.Pickup{}
	db.DbManager().Where("id = ?", pickupId).Find(&pickup)

	if pickup.ID == 0 {
		payload := &PayloadError{
			Errors: "Pickup not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	if permission.OwnerPermissionPickup(claims,&pickup) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}


	//update for pickup
	podPickup := model.PodPickup{}
	podPickup.PickupId = pickupIdInt

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

		fileName := services.UploadS3(fileStream, "pods")
		podPickup.Image = fileName

		//create pod for each order relate pickup
		orders := []*model.Order{}
		db.DbManager().Where("pickup_id = ?", pickupId).Find(&orders)
		for _, order := range orders {

			pod := model.FirstMilePodOrder{}
			pod.OrderId = int(order.ID)
			pod.Image = fileName

			db.DbManager().Create(&pod)
		}

	}

	db.DbManager().Create(&podPickup)

	if podPickup.Image != "" {
		podPickup.Image = helper.GetS3Url(podPickup.Image, "pods")
	}

	payload := &PayloadSuccess{
		Data: &podPickup,
	}
	return c.JSON(http.StatusOK, payload)
}

func DeletePODPickup(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)

	podId := c.Param("id")

	podPickup := model.PodPickup{}
	db.DbManager().Where("id = ?", podId).Find(&podPickup)

	if podPickup.ID == 0 {
		payload := &PayloadError{
			Errors: "podPickup not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}
	pickup := model.Pickup{}
	db.DbManager().Where("id = ?", podPickup.PickupId).Find(&pickup)

	if permission.OwnerPermissionPickup(claims,&pickup) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}

	//delete pod relate this pickup
	orders := []*model.Order{}
	db.DbManager().Where("pickup_id = ?", podPickup.PickupId).Find(&orders)
	for _, order := range orders {

		podOrder := model.FirstMilePodOrder{}
		db.DbManager().Where("order_id = ?", order.ID).Find(&podOrder)
		db.DbManager().Delete(&podOrder)

	}

	db.DbManager().Delete(&podPickup)

	return c.JSON(http.StatusOK, "Deleted")
}
