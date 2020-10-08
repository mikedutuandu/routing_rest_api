package api

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"io"
	"io/ioutil"
	"janio-backend/db"
	"janio-backend/helper"
	"janio-backend/model"
	"janio-backend/permission"
	"janio-backend/services"
	"janio-backend/validator"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func OptimizeDelivery(c echo.Context) error {
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
	errs := validator.OptimizeDeliveryValidator(d)
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

	//update orders of this driver
	//DELIVERY_IN_PROGRESS
	orders := []*model.Order{}
	query := db.DbManager().Order("created_at desc")
	query = query.Where("delivery_date = ?", today)
	query = query.Where("last_mile_driver_id = ? ", DriverId)
	query = query.Where("status IN (?)", []string{"DELIVERY_SCHEDULED", "DELIVERY_IN_PROGRESS"})
	query.Find(&orders)

	for _, order := range orders {
		order.Status = "DELIVERY_IN_PROGRESS"
		order.SolutionOrder = 9999
		db.DbManager().Save(&order)
	}

	//optimize
	//create job
	Job := model.OptimizeJob{}
	Job.DriverId = int(DriverId)
	Job.Status = "PENDING"
	Job.JobType = "DELIVERY"
	Job.CreatedAt = time.Now()
	Job.UpdatedAt = time.Now()
	db.DbManager().Save(&Job)

	jobId := strconv.Itoa(int(Job.ID))

	respJobId := helper.GetOrderSolution(int(DriverId), orders, lat, lng, jobId)
	Job.JobId = respJobId
	db.DbManager().Save(&Job)

	payload := &PayloadSuccess{
		Data: &Job,
	}
	return c.JSON(http.StatusOK, payload)
}
func ScanToWarehouse(c echo.Context) error {
	exOrderId := c.Param("ex_id")

	order := model.Order{}

	//check duplicated
	count := 0
	duplicated := false
	db.DbManager().Table("orders").Where("tracking_no = ?", exOrderId).Count(&count)
	if count > 1 {
		duplicated = true
	}

	db.DbManager().Where("tracking_no = ?", exOrderId).Find(&order)
	if order.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusBadRequest, payload)
	}

	if order.Status == "ORDER_PICKED_UP" || order.Status == "DELIVERY_RESCHEDULED" {

		order.Status = "ORDER_RECEIVED_AT_LOCAL_SORTING_CENTER"
		order.Duplicated = duplicated
		db.DbManager().Save(&order)
		payload := &PayloadSuccess{
			Data: order,
		}
		return c.JSON(http.StatusOK, payload)
	}

	payload := &PayloadError{
		Errors: "Cannot change [" + order.Status + "] to ORDER_RECEIVED_AT_LOCAL_SORTING_CENTER",
	}
	return c.JSON(http.StatusBadRequest, payload)

}
func ScanToDelivery(c echo.Context) error {
	exOrderId := c.Param("ex_id")

	order := model.Order{}

	db.DbManager().Where("tracking_no = ?", exOrderId).Find(&order)
	if order.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusBadRequest, payload)
	}
	if order.Status != "ORDER_RECEIVED_AT_LOCAL_SORTING_CENTER" {
		payload := &PayloadError{
			Errors: "Cannot change [" + order.Status + "] to DELIVERY_SCHEDULED",
		}
		return c.JSON(http.StatusBadRequest, payload)
	}
	order.Status = "DELIVERY_SCHEDULED"
	order.DeliveryDate = time.Now().Format(layoutISO)
	db.DbManager().Save(&order)

	payload := &PayloadSuccess{
		Data: order,
	}

	return c.JSON(http.StatusOK, payload)
}
func DetailOrder(c echo.Context) error {
	orderId := c.Param("id")

	order := model.Order{}

	db.DbManager().Where("id = ?", orderId).Find(&order)
	if order.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)
	if permission.OwnerPermissionDelivery(claims, &order) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}

	firstMilePods := []*model.FirstMilePodOrder{}
	db.DbManager().Where("order_id = ?", orderId).Find(&firstMilePods)

	for _, pod := range firstMilePods {
		pod.Image = helper.GetS3Url(pod.Image, "pods")
	}

	lastMilePods := []*model.LastMilePodOrder{}
	db.DbManager().Where("order_id = ?", orderId).Find(&lastMilePods)

	for _, pod := range lastMilePods {
		pod.Image = helper.GetS3Url(pod.Image, "pods")
	}

	order.FirstMilePods = firstMilePods
	order.LastMilePods = lastMilePods

	order.CustomerSignatureFirstMile = helper.GetS3Url(order.CustomerSignatureFirstMile, "order-signatures")
	order.CustomerSignatureLastMile = helper.GetS3Url(order.CustomerSignatureLastMile, "order-signatures")

	payload := &PayloadSuccess{
		Data: order,
	}

	return c.JSON(http.StatusOK, payload)
}
func UpdateOrder(c echo.Context) error {

	defer c.Request().Body.Close()

	b, err := ioutil.ReadAll(c.Request().Body)
	//err := json.NewDecoder(c.Request().Body).Decode(&dog)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]string)

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	status := d["status"]
	statusNoteStr := d["status_note"]
	driverNote := d["driver_note"]
	rescheduleDateStr := d["reschedule_date"]
	rescheduleDate, _ := time.Parse(layoutISO, rescheduleDateStr)

	//validate valid status
	validStatus := []string{"DELIVERY_IN_PROGRESS", "SUCCESS", "DELIVERY_RESCHEDULED", "DELIVERY_FAILED"}
	if helper.ItemExists(validStatus, status) == false {
		payload := &PayloadError{
			Errors: "Invalid status! status for delivery must one of (DELIVERY_IN_PROGRESS, SUCCESS, DELIVERY_RESCHEDULED, DELIVERY_FAILED)",
		}
		return c.JSON(http.StatusNotFound, payload)
	}
	//find
	orderId := c.Param("id")
	order := model.Order{}
	db.DbManager().Where("id = ?", orderId).Find(&order)
	if order.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)
	if permission.OwnerPermissionDelivery(claims, &order) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}

	newStatus := status
	if rescheduleDateStr != "" {
		newStatus = "DELIVERY_RESCHEDULED"
	}

	//update
	statusNote := helper.BuildStatus(order.StatusNote, newStatus, statusNoteStr)
	order.Status = newStatus
	order.StatusNote = statusNote
	order.DeliveryNote = driverNote
	order.RescheduleDate = rescheduleDate

	db.DbManager().Save(&order)

	payload := &PayloadSuccess{
		Data: order,
	}

	return c.JSON(http.StatusOK, payload)
}
func ListOrder(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*model.JwtCustomClaims)
	isAdmin := claims.User.IsAdmin

	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	status := c.QueryParam("status")
	externalTn := c.QueryParam("external_tn")
	internalTn := c.QueryParam("internal_tn")
	pickupContactName := c.QueryParam("pickup_contact_name")

	FirstMileDriverName := c.QueryParam("first_mile_driver_name")
	LastMileDriverName := c.QueryParam("last_mile_driver_name")

	pickupDate := c.QueryParam("pickup_date")
	deliveryDate := c.QueryParam("delivery_date")
	withDriver := c.QueryParam("with_driver")
	FirstMileDriverIdStr := c.QueryParam("first_mile_driver_id")
	FirstMileDriverId, _ := strconv.Atoi(FirstMileDriverIdStr)

	LastMileDriverIdStr := c.QueryParam("last_mile_driver_id")
	LastMileDriverId, _ := strconv.Atoi(LastMileDriverIdStr)
	listStatusString := []string{}

	consigneeNumber := c.QueryParam("consignee_number")
	consigneeAddress := c.QueryParam("consignee_address")
	pickupAddress := c.QueryParam("pickup_address")
	deliveryAddress := c.QueryParam("delivery_address")
	pickupRegion, _ := strconv.Atoi(c.QueryParam("pickup_region"))
	deliveryRegion, _ := strconv.Atoi(c.QueryParam("delivery_region"))

	if status != "" {
		listStatus := strings.Split(status, ",")
		for _, item := range listStatus {
			listStatusString = append(listStatusString, item)
		}
	}
	// Defaults
	if offset == 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 10
	}

	db := db.DbManager()
	orders := []*model.Order{}
	count := 0
	query := db.Offset(offset).Limit(limit).Order("solution_order asc")
	queryCount := db.Table("orders")
	if len(listStatusString) > 0 {
		query = query.Where("status IN (?)", listStatusString)
		queryCount = queryCount.Where("status IN (?)", listStatusString)
	}
	if pickupDate != "" {
		query = query.Where("DATE(pickup_date) = ?", pickupDate)
		queryCount = queryCount.Where("DATE(pickup_date) = ?", pickupDate)
	}
	if deliveryDate != "" {
		query = query.Where("DATE(delivery_date) = ?", deliveryDate)
		queryCount = queryCount.Where("DATE(delivery_date) = ?", deliveryDate)
	}
	if externalTn != "" {
		query = query.Where("tracking_no LIKE ?", "%"+externalTn+"%")
		queryCount = queryCount.Where("tracking_no LIKE ?", "%"+externalTn+"%")
	}
	if internalTn != "" {
		query = query.Where("order_id LIKE ?", "%"+internalTn+"%")
		queryCount = queryCount.Where("order_id LIKE ?", "%"+internalTn+"%")
	}
	if pickupContactName != "" {
		query = query.Where("pickup_contact_name LIKE ?", "%"+pickupContactName+"%")
		queryCount = queryCount.Where("pickup_contact_name LIKE ?", "%"+pickupContactName+"%")
	}
	if FirstMileDriverName != "" {
		query = query.Where("first_mile_driver_name LIKE ?", "%"+FirstMileDriverName+"%")
		queryCount = queryCount.Where("first_mile_driver_name LIKE ?", "%"+FirstMileDriverName+"%")
	}
	if LastMileDriverName != "" {
		query = query.Where("last_mile_driver_name LIKE ?", "%"+LastMileDriverName+"%")
		queryCount = queryCount.Where("last_mile_driver_name LIKE ?", "%"+LastMileDriverName+"%")
	}

	if FirstMileDriverId != 0 {
		query = query.Where("first_mile_driver_id = ? ", FirstMileDriverId)
		queryCount = queryCount.Where("first_mile_driver_id  = ?", FirstMileDriverId)
	}
	if LastMileDriverId != 0 {
		query = query.Where("last_mile_driver_id = ? ", LastMileDriverId)
		queryCount = queryCount.Where("last_mile_driver_id  = ?", LastMileDriverId)
	}

	if isAdmin == false {
		userId := claims.User.ID
		if userId != 0 {
			query = query.Where("first_mile_driver_id = ? OR last_mile_driver_id = ? ", userId, userId)
			queryCount = queryCount.Where("first_mile_driver_id  = ? OR last_mile_driver_id = ?", userId, userId)
		}
	}

	if consigneeNumber != "" {
		query = query.Where("consignee_number LIKE ?", "%"+consigneeNumber+"%")
		queryCount = queryCount.Where("consignee_number LIKE ?", "%"+consigneeNumber+"%")
	}

	if consigneeAddress != "" {
		query = query.Where("consignee_address LIKE ?", "%"+consigneeAddress+"%")
		queryCount = queryCount.Where("consignee_address LIKE ?", "%"+consigneeAddress+"%")
	}
	if deliveryAddress != "" {
		query = query.Where("consignee_address LIKE ?", "%"+deliveryAddress+"%")
		queryCount = queryCount.Where("consignee_address LIKE ?", "%"+deliveryAddress+"%")
	}

	if pickupAddress != "" {
		query = query.Where("pickup_address LIKE ?", "%"+pickupAddress+"%")
		queryCount = queryCount.Where("pickup_address LIKE ?", "%"+pickupAddress+"%")
	}

	if pickupRegion != 0 {
		query = query.Where("pickup_region = ?", pickupRegion)
		queryCount = queryCount.Where("pickup_region = ?", pickupRegion)
	}

	if deliveryRegion != 0 {
		query = query.Where("delivery_region = ?", deliveryRegion)
		queryCount = queryCount.Where("delivery_region = ?", deliveryRegion)
	}

	query.Find(&orders)
	queryCount.Count(&count)

	if withDriver == "" {
		for index, _ := range orders {
			orders[index].LastMileDriverId = 0
			orders[index].FirstMileDriverId = 0
			orders[index].LastMileDriverName = ""
			orders[index].FirstMileDriverName = ""
		}
	}
	payload := &PayloadSuccess{
		Data: orders,
		Meta: struct {
			TotalRecord int `json:"total_record"`
		}{count},
	}
	return c.JSON(http.StatusOK, payload)
}

func DeleteFirstMilePODOrder(c echo.Context) error {
	podId := c.Param("id")

	pod := model.FirstMilePodOrder{}
	db.DbManager().Where("id = ?", podId).Find(&pod)

	db.DbManager().Delete(&pod)

	return c.JSON(http.StatusOK, "Deleted")
}
func DeleteLastMilePODOrder(c echo.Context) error {
	podId := c.Param("id")

	pod := model.LastMilePodOrder{}
	db.DbManager().Where("id = ?", podId).Find(&pod)

	db.DbManager().Delete(&pod)

	return c.JSON(http.StatusOK, "Deleted")
}
func UploadFirstMilePODOrder(c echo.Context) error {
	orderId := c.Param("id")
	orderIdInt, _ := strconv.Atoi(orderId)

	fileStream, _ := c.FormFile("image")

	pod := model.FirstMilePodOrder{}
	pod.OrderId = orderIdInt

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
		pod.Image = fileName
	}
	db.DbManager().Create(&pod)

	if pod.Image != "" {
		pod.Image = helper.GetS3Url(pod.Image, "pods")
	}
	payload := &PayloadSuccess{
		Data: &pod,
	}
	return c.JSON(http.StatusOK, payload)
}
func UploadLastMilePODOrder(c echo.Context) error {
	orderId := c.Param("id")
	orderIdInt, _ := strconv.Atoi(orderId)

	fileStream, _ := c.FormFile("image")

	pod := model.LastMilePodOrder{}
	pod.OrderId = orderIdInt

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
		pod.Image = fileName
	}
	db.DbManager().Create(&pod)

	if pod.Image != "" {
		pod.Image = helper.GetS3Url(pod.Image, "pods")
	}
	payload := &PayloadSuccess{
		Data: &pod,
	}
	return c.JSON(http.StatusOK, payload)
}

func UploadFirstMileCustomerSignatureOrder(c echo.Context) error {
	orderId := c.Param("id")
	exitedOrder := &model.Order{}
	db.DbManager().Where("id = ?", orderId).First(exitedOrder)
	if exitedOrder.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	fileStream, err := c.FormFile("customer_signature")
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}

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

		//ADD TO ORDER
		exitedOrder.CustomerSignatureFirstMile = fileName

	}
	db.DbManager().Save(exitedOrder)
	exitedOrder.CustomerSignatureFirstMile = helper.GetS3Url(exitedOrder.CustomerSignatureFirstMile, "order-signatures")
	exitedOrder.CustomerSignatureLastMile = helper.GetS3Url(exitedOrder.CustomerSignatureLastMile, "order-signatures")
	payload := &PayloadSuccess{
		Data: &exitedOrder,
	}
	return c.JSON(http.StatusOK, payload)

}
func UploadLastMileCustomerSignatureOrder(c echo.Context) error {
	orderId := c.Param("id")
	exitedOrder := &model.Order{}
	db.DbManager().Where("id = ?", orderId).First(exitedOrder)
	if exitedOrder.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	fileStream, err := c.FormFile("customer_signature")
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}

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

		//ADD TO ORDER
		exitedOrder.CustomerSignatureLastMile = fileName

	}
	db.DbManager().Save(exitedOrder)
	exitedOrder.CustomerSignatureFirstMile = helper.GetS3Url(exitedOrder.CustomerSignatureFirstMile, "order-signatures")
	exitedOrder.CustomerSignatureLastMile = helper.GetS3Url(exitedOrder.CustomerSignatureLastMile, "order-signatures")
	payload := &PayloadSuccess{
		Data: &exitedOrder,
	}
	return c.JSON(http.StatusOK, payload)
}
func DeleteFirstMileCustomerSignatureOrder(c echo.Context) error {
	orderId := c.Param("id")
	exitedOrder := &model.Order{}
	db.DbManager().Where("id = ?", orderId).First(exitedOrder)
	if exitedOrder.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}
	exitedOrder.CustomerSignatureFirstMile = ""
	db.DbManager().Save(exitedOrder)

	return c.JSON(http.StatusOK, "Deleted")
}
func DeleteLastMileCustomerSignatureOrder(c echo.Context) error {
	orderId := c.Param("id")
	exitedOrder := &model.Order{}
	db.DbManager().Where("id = ?", orderId).First(exitedOrder)
	if exitedOrder.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	exitedOrder.CustomerSignatureLastMile = ""
	db.DbManager().Save(exitedOrder)

	return c.JSON(http.StatusOK, "Deleted")
}

func UploadOrder(c echo.Context) (err error) {
	userAuth := c.Get("user").(*jwt.Token)
	claims := userAuth.Claims.(*model.JwtCustomClaims)
	if permission.AdminPermission(claims) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}

	file, err := c.FormFile("file")
	pickupDate := c.FormValue("pickup_date")
	t2, _ := time.Parse(layoutISO, pickupDate)
	t2 = t2.AddDate(0, 0, 1)
	deliveryDate := t2.Format(layoutISO)
	override := c.FormValue("override")
	assignmentType := c.FormValue("assignment_type")
	uploadType := c.FormValue("upload_type")

	//VALIDATE
	if services.IsCsv(file) == false {
		payload := &PayloadError{
			Errors: "file should be csv",
		}
		return c.JSON(http.StatusBadRequest, payload)
	}
	//END VALIDATE

	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Destination
	newName := helper.BuildFileName(file.Filename)
	dst, err := os.Create("assets/orders/" + newName)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	importOrder := model.Import{}
	importOrder.Status = "PENDING"
	importOrder.PickupDate = pickupDate
	importOrder.DeliveryDate = deliveryDate
	importOrder.Override = override
	importOrder.AssignmentType = assignmentType
	importOrder.CreatedAt = time.Now()
	importOrder.UpdatedAt = time.Now()
	importOrder.FileName = newName
	importOrder.Username = claims.User.Username
	importOrder.UploadType = uploadType

	db.DbManager().Create(&importOrder)

	//validate file(first line)
	csvFile, _ := os.Open("assets/orders/" + newName)
	reader := csv.NewReader(bufio.NewReader(csvFile))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	line, _ := reader.Read()
	if len(line) != 55 {
		payload := &PayloadError{
			Errors: "Wrong format upload file",
		}
		return c.JSON(http.StatusBadRequest, payload)
	}

	// Destination
	//listErrRow := helper.ImportOrder(newName,override,pickupDate,deliveryDate)
	//start background job
	go helper.UploadOrder(&importOrder)

	payload := &PayloadSuccess{
		Data: importOrder,
	}

	return c.JSON(http.StatusCreated, payload)
}
func DetailUploadOrder(c echo.Context) error {
	userAuth := c.Get("user").(*jwt.Token)
	claims := userAuth.Claims.(*model.JwtCustomClaims)
	if permission.AdminPermission(claims) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}
	uploadId := c.Param("id")

	orderImport := model.Import{}

	db.DbManager().Where("id = ?", uploadId).Find(&orderImport)
	if orderImport.ID == 0 {
		payload := &PayloadError{
			Errors: "Order Import not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	payload := &PayloadSuccess{
		Data: &orderImport,
	}

	return c.JSON(http.StatusOK, payload)
}
func AdminUpdateOrder(c echo.Context) (err error) {

	userAuth := c.Get("user").(*jwt.Token)
	claims := userAuth.Claims.(*model.JwtCustomClaims)
	if permission.AdminPermission(claims) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}

	orderId := c.Param("id")

	defer c.Request().Body.Close()

	b, err := ioutil.ReadAll(c.Request().Body)
	//err := json.NewDecoder(c.Request().Body).Decode(&dog)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]interface{})

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	//infor
	PickupStart := d["pickup_start"].(string)
	PickupEnd := d["pickup_end"].(string)
	DeliveryStart := d["delivery_start"].(string)
	DeliveryEnd := d["delivery_end"].(string)

	OrderId := d["internal_tn"].(string)
	OrderHeight := d["order_height"].(string)
	OrderWidth := d["order_width"].(string)
	OrderWeight := d["order_weight"].(string)
	OrderLength := d["order_length"].(string)
	OrderLabelUrl := d["order_label_url"].(string)
	PaymentType := d["payment_type"].(string)
	TrackingNo := d["external_tn"].(string)
	AgentApplicationIdId := d["agent_application_id_id"].(string)
	ShipperOrderId := d["shipper_order_id"].(string)
	UploadBatchNo := d["upload_batch_no"].(string)
	Status := d["status"].(string)

	//first mile
	PickupAddress := d["pickup_address"].(string)
	PickupCity := d["pickup_city"].(string)
	PickupContactName := d["pickup_contact_name"].(string)
	PickupContactNumber := d["pickup_contact_number"].(string)
	PickupCountry := d["pickup_country"].(string)
	PickupPostal := d["pickup_postal"].(string)
	PickupProvince := d["pickup_province"].(string)
	PickupState := d["pickup_state"].(string)
	PickupNote := d["pickup_note"].(string)
	FirstMileDriverIdStr := d["first_mile_driver_id"].(string)
	FirstMileDriverId, _ := strconv.Atoi(FirstMileDriverIdStr)
	PickupDate := d["pickup_date"].(string)

	//last mile
	var CodAmtToCollect float64 = -1
	if d["cod_amt_to_collect"] != nil {
		codAmtToCollect, _ := strconv.ParseFloat(d["cod_amt_to_collect"].(string), 64)
		CodAmtToCollect = codAmtToCollect
	}
	ConsigneeAddress := d["consignee_address"].(string)
	ConsigneeCity := d["consignee_city"].(string)
	ConsigneeCountry := d["consignee_country"].(string)
	ConsigneeEmail := d["consignee_email"].(string)
	ConsigneeName := d["consignee_name"].(string)
	ConsigneeNumber := d["consignee_number"].(string)
	ConsigneePostal := d["consignee_postal"].(string)
	ConsigneeProvince := d["consignee_province"].(string)
	ConsigneeState := d["consignee_state"].(string)
	DeliveryNote := d["delivery_note"].(string)
	LastMileDriverIdStr := d["last_mile_driver_id"].(string)
	LastMileDriverId, _ := strconv.Atoi(LastMileDriverIdStr)
	DeliveryDate := d["delivery_date"].(string)

	//get more driver name
	FirstDriver := model.User{}
	db.DbManager().Where("id = ?", FirstMileDriverId).Find(&FirstDriver)

	LastDriver := model.User{}
	db.DbManager().Where("id = ?", LastMileDriverId).Find(&LastDriver)

	//2. Handle pickup
	existedPickup := &model.Pickup{}
	db.DbManager().Where("pickup_contact_number = ? and pickup_date = ? and pickup_address = ?", PickupContactNumber, PickupDate, PickupAddress).First(existedPickup)
	existedPickup.PickupAddress = PickupAddress
	existedPickup.PickupCity = PickupCity
	existedPickup.PickupContactName = PickupContactName
	existedPickup.PickupContactNumber = PickupContactNumber
	existedPickup.PickupCountry = PickupCountry
	existedPickup.PickupPostal = PickupPostal
	existedPickup.PickupProvince = PickupProvince
	existedPickup.PickupState = PickupState
	existedPickup.DriverNote = ""
	existedPickup.StatusNote = ""
	existedPickup.CustomerSignature = ""
	existedPickup.PickupDate = PickupDate
	existedPickup.DriverId = int(FirstMileDriverId)
	newStatusPickup := existedPickup.Status
	if newStatusPickup == "" {
		existedPickup.Status = "SCHEDULED"
	}
	existedPickup.Status = newStatusPickup
	existedPickup.PickupStart = "13:00"
	existedPickup.PickupEnd = "18:00"

	if existedPickup.ID == 0 {
		db.DbManager().Create(existedPickup)
	} else {
		db.DbManager().Save(existedPickup)
	}
	pickupId := existedPickup.ID

	//3. Handle order
	exitedOrder := &model.Order{}
	db.DbManager().Where("id = ?", orderId).First(exitedOrder)
	if exitedOrder.ID == 0 {
		payload := &PayloadError{
			Errors: "Order not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	//infor
	exitedOrder.OrderID = OrderId
	exitedOrder.OrderHeight = OrderHeight
	exitedOrder.OrderWidth = OrderWidth
	exitedOrder.OrderWeight = OrderWeight
	exitedOrder.OrderLength = OrderLength
	exitedOrder.OrderLabelURL = OrderLabelUrl
	exitedOrder.PaymentType = PaymentType
	exitedOrder.TrackingNo = TrackingNo
	exitedOrder.AgentApplicationIDID = AgentApplicationIdId
	exitedOrder.ShipperOrderID = ShipperOrderId
	exitedOrder.UploadBatchNo = UploadBatchNo

	newStatus := Status
	if FirstMileDriverId == 0 {
		newStatus = "ORDER_INFO_RECEIVED"
	}
	exitedOrder.Status = newStatus
	exitedOrder.PickupStart = PickupStart
	exitedOrder.PickupEnd = PickupEnd
	exitedOrder.DeliveryStart = DeliveryStart
	exitedOrder.DeliveryEnd = DeliveryEnd

	//First mile
	exitedOrder.PickupAddress = PickupAddress
	exitedOrder.PickupCity = PickupCity
	exitedOrder.PickupContactName = PickupContactName
	exitedOrder.PickupContactNumber = PickupContactNumber
	exitedOrder.PickupCountry = PickupCountry
	exitedOrder.PickupPostal = PickupPostal
	exitedOrder.PickupProvince = PickupProvince
	exitedOrder.PickupState = PickupState
	exitedOrder.PickupNote = PickupNote
	exitedOrder.FirstMileDriverId = FirstMileDriverId
	exitedOrder.PickupDate = PickupDate

	//Last mile
	CodAmtToCollectStr := strconv.FormatFloat(CodAmtToCollect, 'f', -1, 64)
	if CodAmtToCollectStr == "-1" {
		CodAmtToCollectStr = ""
	}

	exitedOrder.CodAmtToCollect = CodAmtToCollectStr
	exitedOrder.ConsigneeAddress = ConsigneeAddress
	exitedOrder.ConsigneeCity = ConsigneeCity
	exitedOrder.ConsigneeCountry = ConsigneeCountry
	exitedOrder.ConsigneeEmail = ConsigneeEmail
	exitedOrder.ConsigneeName = ConsigneeName
	exitedOrder.ConsigneeNumber = ConsigneeNumber
	exitedOrder.ConsigneePostal = ConsigneePostal
	exitedOrder.ConsigneeProvince = ConsigneeProvince
	exitedOrder.ConsigneeState = ConsigneeState
	exitedOrder.DeliveryNote = DeliveryNote
	exitedOrder.DeliveryDate = DeliveryDate
	exitedOrder.LastMileDriverId = LastMileDriverId

	//add driver name
	exitedOrder.FirstMileDriverName = FirstDriver.DriverName
	exitedOrder.LastMileDriverName = LastDriver.DriverName

	exitedOrder.PickupId = int(pickupId)

	//add time

	db.DbManager().Save(exitedOrder)

	exitedOrder.CustomerSignatureFirstMile = helper.GetS3Url(exitedOrder.CustomerSignatureFirstMile, "order-signatures")
	exitedOrder.CustomerSignatureLastMile = helper.GetS3Url(exitedOrder.CustomerSignatureLastMile, "order-signatures")

	payload := &PayloadSuccess{
		Data: &exitedOrder,
	}
	return c.JSON(http.StatusOK, payload)

}
func AdminCreateOrder(c echo.Context) (err error) {
	userAuth := c.Get("user").(*jwt.Token)
	claims := userAuth.Claims.(*model.JwtCustomClaims)
	if permission.AdminPermission(claims) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}

	defer c.Request().Body.Close()

	b, err := ioutil.ReadAll(c.Request().Body)
	//err := json.NewDecoder(c.Request().Body).Decode(&dog)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]string)

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	//infor
	PickupStart := d["pickup_start"]
	PickupEnd := d["pickup_end"]
	DeliveryStart := d["delivery_start"]
	DeliveryEnd := d["delivery_end"]

	OrderId := d["internal_tn"]
	OrderHeight := d["order_height"]
	OrderWidth := d["order_width"]
	OrderWeight := d["order_weight"]
	OrderLength := d["order_length"]
	OrderLabelUrl := d["order_label_url"]
	PaymentType := d["payment_type"]
	TrackingNo := d["external_tn"]
	AgentApplicationIdId := d["agent_application_id_id"]
	ShipperOrderId := d["shipper_order_id"]
	UploadBatchNo := d["upload_batch_no"]
	Status := d["status"]

	//first mile
	PickupAddress := d["pickup_address"]
	PickupCity := d["pickup_city"]
	PickupContactName := d["pickup_contact_name"]
	PickupContactNumber := d["pickup_contact_number"]
	PickupCountry := d["pickup_country"]
	PickupPostal := d["pickup_postal"]
	PickupProvince := d["pickup_province"]
	PickupState := d["pickup_state"]
	PickupNote := d["pickup_note"]
	FirstMileDriverIdStr := d["first_mile_driver_id"]
	FirstMileDriverId, _ := strconv.Atoi(FirstMileDriverIdStr)
	PickupDate := d["pickup_date"]

	//last mile
	CodAmtToCollect := d["cod_amt_to_collect"]
	ConsigneeAddress := d["consignee_address"]
	ConsigneeCity := d["consignee_city"]
	ConsigneeCountry := d["consignee_country"]
	ConsigneeEmail := d["consignee_email"]
	ConsigneeName := d["consignee_name"]
	ConsigneeNumber := d["consignee_number"]
	ConsigneePostal := d["consignee_postal"]
	ConsigneeProvince := d["consignee_province"]
	ConsigneeState := d["consignee_state"]
	DeliveryNote := d["delivery_note"]
	LastMileDriverIdStr := d["last_mile_driver_id"]
	LastMileDriverId, _ := strconv.Atoi(LastMileDriverIdStr)
	DeliveryDate := d["delivery_date"]

	//get more driver name
	FirstDriver := model.User{}
	db.DbManager().Where("id = ?", FirstMileDriverId).Find(&FirstDriver)

	LastDriver := model.User{}
	db.DbManager().Where("id = ?", LastMileDriverId).Find(&LastDriver)

	//2. Handle pickup
	existedPickup := &model.Pickup{}
	db.DbManager().Where("pickup_contact_number = ? and DATE(pickup_date) = ? and pickup_address = ?", PickupContactNumber, PickupDate, PickupAddress).First(existedPickup)
	existedPickup.PickupAddress = PickupAddress
	existedPickup.PickupCity = PickupCity
	existedPickup.PickupContactName = PickupContactName
	existedPickup.PickupContactNumber = PickupContactNumber
	existedPickup.PickupCountry = PickupCountry
	existedPickup.PickupPostal = PickupPostal
	existedPickup.PickupProvince = PickupProvince
	existedPickup.PickupState = PickupState
	existedPickup.DriverNote = ""
	existedPickup.StatusNote = ""
	existedPickup.CustomerSignature = ""
	existedPickup.PickupDate = PickupDate
	existedPickup.DriverId = FirstMileDriverId
	newStatusPickup := existedPickup.Status
	if newStatusPickup == "" {
		existedPickup.Status = "SCHEDULED"
	}
	existedPickup.Status = newStatusPickup
	existedPickup.PickupStart = "13:00"
	existedPickup.PickupEnd = "18:00"

	if existedPickup.ID == 0 {
		db.DbManager().Create(existedPickup)
	} else {
		db.DbManager().Save(existedPickup)
	}
	pickupId := existedPickup.ID

	//3. Handle order
	order := &model.Order{}

	//infor
	order.OrderID = OrderId
	order.OrderHeight = OrderHeight
	order.OrderWidth = OrderWidth
	order.OrderWeight = OrderWeight
	order.OrderLength = OrderLength
	order.OrderLabelURL = OrderLabelUrl
	order.PaymentType = PaymentType
	order.TrackingNo = TrackingNo
	order.AgentApplicationIDID = AgentApplicationIdId
	order.ShipperOrderID = ShipperOrderId
	order.UploadBatchNo = UploadBatchNo

	newStatus := Status
	if FirstMileDriverId == 0 {
		newStatus = "ORDER_INFO_RECEIVED"
	}
	order.Status = newStatus
	order.PickupStart = PickupStart
	order.PickupEnd = PickupEnd
	order.DeliveryStart = DeliveryStart
	order.DeliveryEnd = DeliveryEnd

	//First mile
	order.PickupAddress = PickupAddress
	order.PickupCity = PickupCity
	order.PickupContactName = PickupContactName
	order.PickupContactNumber = PickupContactNumber
	order.PickupCountry = PickupCountry
	order.PickupPostal = PickupPostal
	order.PickupProvince = PickupProvince
	order.PickupState = PickupState
	order.PickupNote = PickupNote
	order.FirstMileDriverId = FirstMileDriverId
	order.PickupDate = PickupDate

	//Last mile
	CodAmtToCollectStr := fmt.Sprintf("%f", CodAmtToCollect)

	order.CodAmtToCollect = CodAmtToCollectStr
	order.ConsigneeAddress = ConsigneeAddress
	order.ConsigneeCity = ConsigneeCity
	order.ConsigneeCountry = ConsigneeCountry
	order.ConsigneeEmail = ConsigneeEmail
	order.ConsigneeName = ConsigneeName
	order.ConsigneeNumber = ConsigneeNumber
	order.ConsigneePostal = ConsigneePostal
	order.ConsigneeProvince = ConsigneeProvince
	order.ConsigneeState = ConsigneeState
	order.DeliveryNote = DeliveryNote
	order.DeliveryDate = DeliveryDate
	order.LastMileDriverId = LastMileDriverId

	//add driver name
	order.FirstMileDriverName = FirstDriver.DriverName
	order.LastMileDriverName = LastDriver.DriverName

	order.PickupId = int(pickupId)

	db.DbManager().Create(order)

	order.CustomerSignatureFirstMile = helper.GetS3Url(order.CustomerSignatureFirstMile, "order-signatures")
	order.CustomerSignatureLastMile = helper.GetS3Url(order.CustomerSignatureLastMile, "order-signatures")

	payload := &PayloadSuccess{
		Data: &order,
	}
	return c.JSON(http.StatusOK, payload)

}
func ResetOrder(c echo.Context) error {

	db.DbManager().Exec("TRUNCATE TABLE orders")
	db.DbManager().Exec("TRUNCATE TABLE pickups")
	db.DbManager().Exec("TRUNCATE TABLE pod_pickups")
	db.DbManager().Exec("TRUNCATE TABLE first_mile_pod_orders")
	db.DbManager().Exec("TRUNCATE TABLE last_mile_pod_orders")

	return c.JSON(http.StatusOK, "Done")
}
func UpdateBulkOrder(c echo.Context) error {

	defer c.Request().Body.Close()

	b, err := ioutil.ReadAll(c.Request().Body)
	//err := json.NewDecoder(c.Request().Body).Decode(&dog)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]string)

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	userAuth := c.Get("user").(*jwt.Token)
	claims := userAuth.Claims.(*model.JwtCustomClaims)
	if permission.AdminPermission(claims) == false {
		payload := &PayloadError{
			Errors: "Permission denied",
		}
		return c.JSON(http.StatusForbidden, payload)
	}

	UpdateData := map[string]interface{}{}

	PickupDate := d["pickup_date"]
	if PickupDate != "" {
		UpdateData["pickup_date"] = PickupDate
	}

	DeliveryDate := d["delivery_date"]
	if DeliveryDate != "" {
		UpdateData["delivery_date"] = DeliveryDate
	}

	PickupStart := d["pickup_start"]
	if PickupStart != "" {
		UpdateData["pickup_start"] = PickupStart
	}
	PickupEnd := d["pickup_end"]
	if PickupEnd != "" {
		UpdateData["pickup_end"] = PickupEnd
	}
	DeliveryStart := d["delivery_start"]
	if DeliveryStart != "" {
		UpdateData["delivery_start"] = DeliveryStart
	}
	DeliveryEnd := d["delivery_end"]
	if DeliveryEnd != "" {
		UpdateData["delivery_end"] = DeliveryEnd
	}
	Status := d["status"]
	if Status != "" {
		UpdateData["status"] = Status
	}

	FirstMileDriverIdStr := d["first_mile_driver_id"]
	if FirstMileDriverIdStr != "" {
		FirstMileDriverId, _ := strconv.Atoi(FirstMileDriverIdStr)
		//get more driver name
		FirstDriver := model.User{}
		db.DbManager().Where("id = ?", FirstMileDriverId).Find(&FirstDriver)

		UpdateData["first_mile_driver_id"] = FirstMileDriverId
		UpdateData["first_mile_driver_name"] = FirstDriver.DriverName
	}

	LastMileDriverIdStr := d["last_mile_driver_id"]
	if LastMileDriverIdStr != "" {
		LastMileDriverId, _ := strconv.Atoi(LastMileDriverIdStr)
		//get more driver name
		LastDriver := model.User{}
		db.DbManager().Where("id = ?", LastMileDriverId).Find(&LastDriver)

		UpdateData["last_mile_driver_id"] = LastMileDriverId
		UpdateData["last_mile_driver_name"] = LastDriver.DriverName
	}

	ListOrderIdStr := d["list_order_id"]
	ListOrderIdArr := strings.Split(ListOrderIdStr, ",")

	db.DbManager().Table("orders").Where("id IN (?)", ListOrderIdArr).Updates(UpdateData)

	orders := []*model.Order{}
	db.DbManager().Where("id IN (?)", ListOrderIdArr).Find(&orders)

	for _, order := range orders {
		pickup := model.Pickup{}
		db.DbManager().Where("pickup_date = ? and driver_id = ?", order.PickupDate, order.FirstMileDriverId).First(&pickup)
		order.PickupId = int(pickup.ID)
		db.DbManager().Save(&order)

	}

	payload := &PayloadSuccess{
		Data: &orders,
	}
	return c.JSON(http.StatusOK, payload)
}

func ListImport(c echo.Context) error {
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

	geocodeLargerThan := c.QueryParam("geocode_larger_than")
	geofenceLargerThan := c.QueryParam("geofence_larger_than")
	username := c.QueryParam("username")
	status := c.QueryParam("status")
	submitDateFrom := c.QueryParam("submit_date_from")
	submitDateTo := c.QueryParam("submit_date_to")
	uploadedAt := c.QueryParam("uploaded_at")
	id := c.QueryParam("id")

	// Defaults
	if offset == 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 10
	}

	db := db.DbManager()
	Import := []*model.Import{}
	count := 0
	query := db.Offset(offset).Limit(limit).Order("created_at desc")
	queryCount := db.Table("imports")
	if geocodeLargerThan != "" {
		query = query.Where("geocode_time > ?", geocodeLargerThan)
		queryCount = queryCount.Where("geocode_time > ?", geocodeLargerThan)
	}
	if geofenceLargerThan != "" {
		query = query.Where("geofence_time > ?", geofenceLargerThan)
		queryCount = queryCount.Where("geofence_time > ?", geofenceLargerThan)
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
		queryCount = queryCount.Where("username LIKE ?", "%"+username+"%")
	}

	if status != "" {
		query = query.Where("status = ?", status)
		queryCount = queryCount.Where("status = ?", status)
	}
	if submitDateFrom != "" {
		query = query.Where("DATE(created_at) >= ?", submitDateFrom)
		queryCount = queryCount.Where("DATE(created_at) >= ?", submitDateFrom)
	}

	if submitDateTo != "" {
		query = query.Where("DATE(created_at) <= ?", submitDateTo)
		queryCount = queryCount.Where("DATE(created_at) <= ?", submitDateTo)
	}

	if uploadedAt != "" {
		query = query.Where("DATE(created_at) = ?", uploadedAt)
		queryCount = queryCount.Where("DATE(created_at) = ?", uploadedAt)
	}
	if id != "" {
		query = query.Where("id = ?", id)
		queryCount = queryCount.Where("id = ?", id)
	}

	query.Find(&Import)
	queryCount.Count(&count)

	payload := &PayloadSuccess{
		Data: Import,
		Meta: struct {
			TotalRecord int `json:"total_record"`
		}{count},
	}
	return c.JSON(http.StatusOK, payload)
}
