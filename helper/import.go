package helper

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"janio-backend/config"
	"janio-backend/constant"
	"janio-backend/db"
	"janio-backend/model"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func GeocodePickup(orderImport *model.Import) {
	//update lat, lng for pickup

	for {
		pickups := []*model.Pickup{}
		db.DbManager().Limit(50).Where("pickup_date = ? and add_lat_lng <> ? and add_lat_lng <> ?", orderImport.PickupDate, "PENDING", "DONE").Find(&pickups)
		if len(pickups) == 0 {
			break
		}
		GetPickupLatLng(pickups)

	}
}
func GeocodeDelivery(orderImport *model.Import) {
	//update lat, lng for delivery
	for {
		orders := []*model.Order{}
		db.DbManager().Limit(50).Where("delivery_date = ? and add_lat_lng <> ? and add_lat_lng <> ?", orderImport.DeliveryDate, "PENDING", "DONE").Find(&orders)
		if len(orders) == 0 {
			break
		}
		GetOrderLatLng(orders)
	}
}

func GeofenceRegion(orderImport *model.Import) {
	//update region
	for {
		orders := []*model.Order{}
		db.DbManager().Limit(50).Where("delivery_date = ? and add_region <> ?", orderImport.DeliveryDate, "DONE").Find(&orders)
		if len(orders) == 0 {
			break
		}
		if orderImport.AssignmentType == "geofencing" {
			GetOrderRegionForPickupAndDelivery(orders)
		} else {
			GetOrderRegionForPickupAndDeliveryBaseOn2Digit(orders)
		}
	}
}
func UpdateDriverForPickups(orderImport *model.Import) {
	//update driver id for pickup
	pickups := []*model.Pickup{}
	db.DbManager().Limit(5000).Where("pickup_date = ? and driver_id = ?", orderImport.PickupDate, 0).Find(&pickups)

	for _, pickup := range pickups {
		order := model.Order{}
		db.DbManager().Where("pickup_date =? AND pickup_address= ?", pickup.PickupDate, pickup.PickupAddress).Find(&order)
		pickup.DriverId = order.FirstMileDriverId
		db.DbManager().Save(&pickup)
	}
}

func UploadOrder(orderImport *model.Import) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	//import
	rep, numOrder, numberFailOrders := ImportOrder(int(orderImport.ID), orderImport.FileName, orderImport.Override, orderImport.PickupDate, orderImport.DeliveryDate, orderImport.AssignmentType, orderImport.UploadType)

	resStr, _ := json.Marshal(rep)
	orderImport.ErrorData = string(resStr)
	orderImport.GeocodePickupStatus = "PENDING"
	orderImport.GeocodeDeliveryStatus = "PENDING"
	orderImport.StartGeocode = time.Now().Unix()
	orderImport.NumberOrder = numOrder - numberFailOrders
	if numOrder == numberFailOrders {
		orderImport.Status = "SUCCESS"
	}

	db.DbManager().Save(&orderImport)

	//update lat, lng

	GeocodePickup(orderImport)

	GeocodeDelivery(orderImport)

}

func UpdateNumberOrderForEachPickup(orderImport *model.Import) {
	//update driver id for pickup
	pickups := []*model.Pickup{}
	db.DbManager().Limit(5000).Where("import_id = ?", orderImport.ID).Find(&pickups)
	for _, pickup := range pickups {
		count := 0
		db.DbManager().Table("orders").Where("pickup_id = ?", pickup.ID).Count(&count)
		pickup.TotalNumberOrder = count
		pickup.TotalNumberOrderPickup = 0
		db.DbManager().Save(&pickup)
	}
}

func ImportOrder(importId int, newName string, override string, pickupDate string, deliveryDate string, assignmentType string, uploadType string) ([]map[string]interface{}, int, int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()
	//insert db
	csvFile, _ := os.Open("assets/orders/" + newName)
	reader := csv.NewReader(bufio.NewReader(csvFile))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	index := 1
	listErrRow := []map[string]interface{}{}
	listOrderIdExist := []string{}
	numberOrders := 0
	numberFailOrders := 0
	for {
		line, err := reader.Read()

		fmt.Println("VALIDATE LEN :", len(line))
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}

		orderId := line[0]
		TrackerStatusCode := line[49]
		if orderId != "Order ID" && orderId != "?Order ID" {
			numberOrders = numberOrders + 1
			index = index + 1
			//validate upload
			isEmptyRow := IsEmptyRow(line)
			if isEmptyRow == true {
				continue
			}
			canImport, _, errMsg := ValidateLineOrder(index, line, assignmentType)
			//end validate

			if canImport == true {

				//1. Get data from line
				CodAmtToCollect := line[4]
				PickupPostal := line[32]
				PickupAddress := line[27]
				PickupCity := line[28]
				PickupContactName := line[29]
				PickupContactNumber := line[30]
				PickupCountry := line[31]
				PickupProvince := line[33]
				PickupState := line[34]
				ConsigneeAddress := line[5]
				ConsigneeCity := line[6]
				ConsigneeCountry := line[7]
				ConsigneeEmail := line[8]
				ConsigneeName := line[9]
				ConsigneeNumber := line[10]
				ConsigneePostal := line[11]
				ConsigneeProvince := line[12]
				ConsigneeState := line[13]
				DeliveryNote := line[15]
				//TrackerStatusCode := line[49]
				TrackingNo := line[51]
				UploadBatchNo := line[53]

				AgentApplicationIDID := line[2]
				OrderHeight := line[21]
				OrderLabelURL := line[22]
				OrderLength := line[23]
				OrderWeight := line[24]
				OrderWidth := line[25]
				PaymentType := line[26]
				ShipperOrderID := line[40]
				CashOnDelivery, _ := strconv.ParseFloat(CodAmtToCollect, 64)
				CashOnDeliveryCurrency := ""

				//2. Handle pickup
				existedPickup := &model.Pickup{}
				db.DbManager().Where("pickup_contact_number = ? and pickup_date = ? and pickup_address = ?", PickupContactNumber, pickupDate, PickupAddress).First(existedPickup)
				pickupId := existedPickup.ID

				existedPickup.PickupAddress = PickupAddress
				existedPickup.PickupCity = PickupCity
				existedPickup.PickupContactName = PickupContactName
				existedPickup.PickupContactNumber = PickupContactNumber
				existedPickup.PickupCountry = PickupCountry
				existedPickup.PickupPostal = PickupPostal
				existedPickup.PickupProvince = PickupProvince
				existedPickup.PickupState = PickupState
				if uploadType == "delivery" {
					existedPickup.Status = "COMPLETED"
				} else {
					existedPickup.Status = "SCHEDULED"
				}
				existedPickup.DriverNote = ""
				existedPickup.StatusNote = ""
				existedPickup.CustomerSignature = ""
				existedPickup.PickupDate = pickupDate
				existedPickup.DriverId = 0

				existedPickup.PickupStart = "13:00"
				existedPickup.PickupEnd = "18:00"
				existedPickup.ImportId = importId
				existedPickup.AddLatLng = "NEW"

				if existedPickup.ID == 0 {
					db.DbManager().Create(existedPickup)
				} else {
					db.DbManager().Save(existedPickup)
				}

				pickupId = existedPickup.ID
				fmt.Println("IMPORT PICKUP: ", pickupId)

				//3. Handle order
				exitedOrder := &model.Order{}
				db.DbManager().Where("order_id = ?", orderId).First(exitedOrder)

				exitedOrder.OrderID = orderId
				exitedOrder.TrackingNo = TrackingNo
				exitedOrder.PickupAddress = PickupAddress
				exitedOrder.PickupCity = PickupCity
				exitedOrder.PickupContactName = PickupContactName
				exitedOrder.PickupContactNumber = PickupContactNumber
				exitedOrder.PickupCountry = PickupCountry
				exitedOrder.PickupPostal = PickupPostal
				exitedOrder.PickupProvince = PickupProvince
				exitedOrder.PickupState = PickupState
				exitedOrder.PickupDate = pickupDate
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
				exitedOrder.PickupId = int(pickupId)
				exitedOrder.DeliveryDate = deliveryDate
				exitedOrder.CashOnDelivery = float32(CashOnDelivery)
				exitedOrder.CashOnDeliveryCurrency = CashOnDeliveryCurrency
				exitedOrder.CodAmtToCollect = CodAmtToCollect
				exitedOrder.AgentApplicationIDID = AgentApplicationIDID
				exitedOrder.OrderHeight = OrderHeight
				exitedOrder.OrderLabelURL = OrderLabelURL
				exitedOrder.OrderLength = OrderLength
				exitedOrder.OrderWeight = OrderWeight
				exitedOrder.OrderWidth = OrderWidth
				exitedOrder.PaymentType = PaymentType
				exitedOrder.ShipperOrderID = ShipperOrderID
				exitedOrder.TrackerStatusCode = TrackerStatusCode
				exitedOrder.UploadBatchNo = UploadBatchNo

				exitedOrder.DeliveryStart = "09:00"
				exitedOrder.DeliveryEnd = "18:00"
				exitedOrder.PickupStart = "13:00"
				exitedOrder.PickupEnd = "18:00"
				exitedOrder.ImportId = importId
				exitedOrder.AddLatLng = "NEW"
				if uploadType == "pickup" {
					exitedOrder.Status = "ORDER_INFO_RECEIVED"
				} else if uploadType == "delivery" {
					exitedOrder.Status = "ORDER_RECEIVED_AT_LOCAL_SORTING_CENTER"
				} else {
					exitedOrder.Status = "ORDER_INFO_RECEIVED"
				}

				if override == "true" {
					if exitedOrder.ID == 0 {
						db.DbManager().Create(exitedOrder)
					} else {
						db.DbManager().Save(exitedOrder)
					}
					fmt.Println("IMPORT ORDER: ", exitedOrder.ID)
				} else {
					if exitedOrder.ID != 0 {
						//error here
						listOrderIdExist = append(listOrderIdExist, exitedOrder.OrderID)
					} else {
						db.DbManager().Create(exitedOrder)
					}
				}

			} else {
				listErrRow = append(listErrRow, errMsg)
				numberFailOrders = numberFailOrders + 1
			}

		}
	}

	if len(listOrderIdExist) != 0 {
		a := make(map[string]interface{})
		a["existed"] = strings.Join(listOrderIdExist, ",")
		listErrRow = append(listErrRow, a)
	}

	return listErrRow, numberOrders, numberFailOrders
}

func GetOrderLatLng(orders []*model.Order) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	data := map[string]interface{}{
		"callback_url": config.GetConfig().SERVER_HOST + "/api/update-lat-lng-order/",
	}

	places := map[string]interface{}{}
	for _, order := range orders {
		order.AddLatLng = "PENDING"
		db.DbManager().Save(&order)
		pickupKey := strconv.Itoa(int(order.ID)) + "_pickup"
		deliveryKey := strconv.Itoa(int(order.ID)) + "_delivery"

		pickupAddress := order.PickupAddress
		if order.PickupCity != "" {
			pickupAddress = pickupAddress + ", " + order.PickupCity
		}
		if order.PickupPostal != "" {
			pickupAddress = pickupAddress + ", " + order.PickupPostal
		}
		if order.PickupCountry != "" {
			pickupAddress = pickupAddress + ", " + order.PickupCountry
		}

		consigneeAddress := order.ConsigneeAddress
		if order.ConsigneeCity != "" {
			consigneeAddress = consigneeAddress + ", " + order.ConsigneeCity
		}
		if order.ConsigneePostal != "" {
			consigneeAddress = consigneeAddress + ", " + order.ConsigneePostal
		}
		if order.ConsigneeCountry != "" {
			consigneeAddress = consigneeAddress + ", " + order.ConsigneeCountry
		}

		places[pickupKey] = map[string]string{
			"address": pickupAddress,
		}
		places[deliveryKey] = map[string]string{
			"address": consigneeAddress,
		}

	}

	data["places"] = places

	dataJsonByte, _ := json.Marshal(data)
	dataJson := string(dataJsonByte)
	fmt.Println(dataJson)

	//call
	client := &http.Client{}

	req, _ := http.NewRequest("POST", "http://janio-geocode.herokuapp.com/api/v1/geocode", bytes.NewBuffer(dataJsonByte))

	req.Header.Add("bearer", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJfaWQiOiI1YjMwNTA0NjBmM2Q1ODdmYWIwYTgxMGUiLCJpYXQiOjE1Mjk4OTI5MzR9.QlmVPRt1-TuxJNQE3B-aG0Qh7OUUIR5qdqd-VWOcD4M")
	req.Header.Set("Content-Type", "application/json")
	_, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}

	fmt.Println(dataJson)

}
func GetPickupLatLng(pickups []*model.Pickup) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	//prepare data
	data := map[string]interface{}{
		"callback_url": config.GetConfig().SERVER_HOST + "/api/update-lat-lng-pickup/",
	}
	//loop places
	places := map[string]interface{}{}
	for _, pickup := range pickups {
		pickup.AddLatLng = "PENDING"
		db.DbManager().Save(&pickup)

		key := strconv.Itoa(int(pickup.ID))

		pickupAddress := pickup.PickupAddress
		if pickup.PickupCity != "" {
			pickupAddress = pickupAddress + ", " + pickup.PickupCity
		}
		if pickup.PickupPostal != "" {
			pickupAddress = pickupAddress + ", " + pickup.PickupPostal
		}
		if pickup.PickupCountry != "" {
			pickupAddress = pickupAddress + ", " + pickup.PickupCountry
		}
		places[key] = map[string]string{
			"address": pickupAddress,
		}
	}
	//add places to data
	data["places"] = places

	dataJsonByte, _ := json.Marshal(data)
	dataJson := string(dataJsonByte)
	fmt.Println(dataJson)

	//call
	client := &http.Client{}

	req, _ := http.NewRequest("POST", "http://janio-geocode.herokuapp.com/api/v1/geocode", bytes.NewBuffer(dataJsonByte))

	req.Header.Add("bearer", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJfaWQiOiI1YjMwNTA0NjBmM2Q1ODdmYWIwYTgxMGUiLCJpYXQiOjE1Mjk4OTI5MzR9.QlmVPRt1-TuxJNQE3B-aG0Qh7OUUIR5qdqd-VWOcD4M")
	req.Header.Set("Content-Type", "application/json")
	_, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}

	fmt.Println(dataJson)

}

func GetPickupSolution(driverId int, pickups []*model.Pickup, lat float64, lng float64, jobId string) string {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	driver := model.User{}
	db.DbManager().Where("id = ?", driverId).Find(&driver)
	//prepare data
	data := map[string]interface{}{
		"callback_url": config.GetConfig().SERVER_HOST + "/api/update-position-pickup?job_id=" + jobId,
	}
	//loop places
	fleet := map[string]interface{}{}
	visits := map[string]interface{}{}
	options := map[string]interface{}{
		"polylines": true,
	}
	breakItem := map[string]string{
		"id":    "break",
		"start": "12:00",
		"end":   "13:00",
	}
	breaks := []map[string]string{breakItem}

	Weight, _ := strconv.ParseFloat(driver.Weight, 32)
	Volume, _ := strconv.ParseFloat(driver.Volume, 32)

	if Weight != 0 && Volume != 0 {
		fleet["driver"] = map[string]interface{}{
			"start_location": map[string]interface{}{
				"id":   "driver",
				"name": driver.DriverName,
				"lat":  lat,
				"lng":  lng,
			},
			"shift_start": shiftStartOrNow(driver.ShiftPickupStart, driver.Timezone),
			"shift_end":   driver.ShiftPickupEnd,
			"breaks":      breaks,
			"capacity": map[string]interface{}{
				"weight": driver.Weight,
				"volume": driver.Volume,
			},
		}
	} else {
		fleet["driver"] = map[string]interface{}{
			"start_location": map[string]interface{}{
				"id":   "driver",
				"name": driver.DriverName,
				"lat":  lat,
				"lng":  lng,
			},
			"shift_start": shiftStartOrNow(driver.ShiftPickupStart, driver.Timezone),
			"shift_end":   driver.ShiftPickupEnd,
			"breaks":      breaks,
		}
	}

	for _, pickup := range pickups {
		key := strconv.Itoa(int(pickup.ID))

		visits[key] = map[string]interface{}{
			"duration": 10,
			"start":    shiftStartOrNow(pickup.PickupStart, driver.Timezone),
			"end":      pickup.PickupEnd,
			"location": map[string]interface{}{
				"name": pickup.PickupAddress,
				"lat":  pickup.PickupLat,
				"lng":  pickup.PickupLng,
			},
		}
	}
	//add places to data
	data["fleet"] = fleet
	data["visits"] = visits
	data["options"] = options

	dataJsonByte, _ := json.Marshal(data)
	dataJson := string(dataJsonByte)
	fmt.Println("PICKUP SOLUTION:", dataJson)

	//call
	client := &http.Client{}

	req, _ := http.NewRequest("POST", "https://routing-engine.afi.io/vrp-long", bytes.NewBuffer(dataJsonByte))

	req.Header.Add("access_token", "xGXhLeVrnRNNAVcNAUZgbttEBySyaLIOrwopriUC")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
		return ""
	}

	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	respJobId, _ := result["job_id"].(string)
	return respJobId
}

func GetOrderSolution(driverId int, orders []*model.Order, lat float64, lng float64, jobId string) string {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	driver := model.User{}
	db.DbManager().Where("id = ?", driverId).Find(&driver)
	//prepare data
	data := map[string]interface{}{
		"callback_url": config.GetConfig().SERVER_HOST + "/api/update-position-order?job_id=" + jobId,
	}
	//loop places
	fleet := map[string]interface{}{}
	visits := map[string]interface{}{}
	options := map[string]interface{}{
		"polylines": true,
	}
	breakItem := map[string]string{
		"id":    "break",
		"start": "12:00",
		"end":   "13:00",
	}
	breaks := []map[string]string{breakItem}
	Weight, _ := strconv.ParseFloat(driver.Weight, 32)
	Volume, _ := strconv.ParseFloat(driver.Volume, 32)

	if Weight != 0 && Volume != 0 {
		fleet["driver"] = map[string]interface{}{
			"start_location": map[string]interface{}{
				"id":   "driver",
				"name": driver.DriverName,
				"lat":  lat,
				"lng":  lng,
			},
			"shift_start": shiftStartOrNow(driver.ShiftDeliveryStart, driver.Timezone),
			"shift_end":   driver.ShiftDeliveryEnd,
			"breaks":      breaks,
			"capacity": map[string]interface{}{
				"weight": driver.Weight,
				"volume": driver.Volume,
			},
		}
	} else {
		fleet["driver"] = map[string]interface{}{
			"start_location": map[string]interface{}{
				"id":   "driver",
				"name": driver.DriverName,
				"lat":  lat,
				"lng":  lng,
			},
			"shift_start": shiftStartOrNow(driver.ShiftDeliveryStart, driver.Timezone),
			"shift_end":   driver.ShiftDeliveryEnd,
			"breaks":      breaks,
		}
	}

	for _, order := range orders {
		key := strconv.Itoa(int(order.ID))

		OrderWeight, _ := strconv.ParseFloat(order.OrderWeight, 32)
		OrderWidth, _ := strconv.ParseFloat(order.OrderWidth, 32)
		OrderHeight, _ := strconv.ParseFloat(order.OrderHeight, 32)
		OrderLength, _ := strconv.ParseFloat(order.OrderLength, 32)
		if OrderWeight != 0 && OrderHeight != 0 && OrderWidth != 0 && OrderLength != 0 {
			visits[key] = map[string]interface{}{
				"duration": 10,
				"start":    shiftStartOrNow(order.DeliveryStart, driver.Timezone),
				"end":      order.DeliveryEnd,
				"location": map[string]interface{}{
					"name": order.ConsigneeAddress,
					"lat":  order.DeliveryLat,
					"lng":  order.DeliveryLng,
				},
				"load": map[string]interface{}{
					"weight": OrderWeight,
					"volume": OrderWidth * OrderHeight * OrderLength,
				},
			}
		} else {
			visits[key] = map[string]interface{}{
				"duration": 10,
				"start":    shiftStartOrNow(order.DeliveryStart, driver.Timezone),
				"end":      order.DeliveryEnd,
				"location": map[string]interface{}{
					"name": order.ConsigneeAddress,
					"lat":  order.DeliveryLat,
					"lng":  order.DeliveryLng,
				},
			}
		}
	}
	//add places to data
	data["fleet"] = fleet
	data["visits"] = visits
	data["options"] = options

	dataJsonByte, _ := json.Marshal(data)
	dataJson := string(dataJsonByte)
	fmt.Println("ORDER SOLUTION:", dataJson)

	client := &http.Client{}

	req, _ := http.NewRequest("POST", "https://routing-engine.afi.io/vrp-long", bytes.NewBuffer(dataJsonByte))
	req.Header.Add("access_token", "xGXhLeVrnRNNAVcNAUZgbttEBySyaLIOrwopriUC")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
		return ""
	}

	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	respJobId, _ := result["job_id"].(string)
	return respJobId
}

func shiftStartOrNow(shiftStart string, timezone string) string {
	// comment for testing, production please uncomment

	loc, _ := time.LoadLocation(timezone)
	t := time.Now().In(loc)
	hourNow := t.Format(constant.TIME_HOUR_LAYOUT)
	if hourNow > shiftStart {
		return hourNow
	}

	return shiftStart
}

func GetOrderRegionForPickupAndDelivery(orders []*model.Order) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	data := map[string]interface{}{}

	places := map[string]interface{}{}
	for _, order := range orders {
		pickupKey := strconv.Itoa(int(order.ID)) + "_pickup"
		deliveryKey := strconv.Itoa(int(order.ID)) + "_delivery"

		places[pickupKey] = map[string]float64{
			"latitude":  order.PickupLat,
			"longitude": order.PickupLng,
		}
		places[deliveryKey] = map[string]float64{
			"latitude":  order.DeliveryLat,
			"longitude": order.DeliveryLng,
		}

	}

	data["places"] = places

	dataJsonByte, _ := json.Marshal(data)
	dataJson := string(dataJsonByte)
	fmt.Println(dataJson)

	//call
	client := &http.Client{}

	req, _ := http.NewRequest("POST", "https://afi-geofence.herokuapp.com/geofence", bytes.NewBuffer(dataJsonByte))

	//req.Header.Add("bearer", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJfaWQiOiI1YjMwNTA0NjBmM2Q1ODdmYWIwYTgxMGUiLCJpYXQiOjE1Mjk4OTI5MzR9.QlmVPRt1-TuxJNQE3B-aG0Qh7OUUIR5qdqd-VWOcD4M")
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}

	//get data from body

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
	}
	d := make(map[string]interface{})

	err = json.Unmarshal(b, &d)
	if err != nil {
	}

	m := d["places"].(map[string]interface{})
	for k, v := range m {

		dataOrder := strings.Split(k, "_")
		orderId := dataOrder[0]
		typeAddress := dataOrder[1]

		n := v.(map[string]interface{})
		region := "-1"
		if n["region"] != nil {
			region = n["region"].(string)
		}

		region = trimLeadingZeroes(region)

		fmt.Println("region:", region)
		fmt.Println("id:", orderId)
		fmt.Println("typeAddress:", typeAddress)

		order := model.Order{}
		region = trimLeadingZeroes(region)
		db.DbManager().Where("id = ?", orderId).Find(&order)
		if typeAddress == "pickup" {
			FirstMileDriverId, FirstMileDriverName := AssignDriver(region, order.PickupCountry)
			order.PickupRegion = region
			order.FirstMileDriverId = int(FirstMileDriverId)
			order.FirstMileDriverName = FirstMileDriverName
		} else {
			LastMileDriverId, LastMileDriverName := AssignDriver(region, order.ConsigneeCountry)
			order.DeliveryRegion = region
			order.LastMileDriverId = int(LastMileDriverId)
			order.LastMileDriverName = LastMileDriverName
		}

		if order.PickupRegion != "" && order.DeliveryRegion != "" {
			order.AddRegion = "DONE"
			if order.Status == "ORDER_INFO_RECEIVED" && order.FirstMileDriverId != 0 {
				order.Status = "PICK_UP_SCHEDULED"
			}
			if order.Status == "ORDER_RECEIVED_AT_LOCAL_SORTING_CENTER" && order.LastMileDriverId != 0 {
				order.Status = "DELIVERY_SCHEDULED"
			}
		}

		db.DbManager().Save(&order)

	}

	fmt.Println(dataJson)
	fmt.Println("res:", d)

}

func GetOrderRegionForPickupAndDeliveryBaseOn2Digit(orders []*model.Order) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	for _, order := range orders {
		//first
		if order.PickupCountry == "Singapore" {
			firstRegion := order.PickupPostal
			if len(firstRegion) == 5 {
				firstRegion = "0" + firstRegion
			}
			firstRegion = trimLeadingZeroes(firstRegion[0:2])
			fmt.Println("firstRegion:", firstRegion)
			FirstMileDriverId, FirstMileDriverName := AssignDriver(firstRegion, order.PickupCountry)
			fmt.Println("FirstMileDriverId:", FirstMileDriverId)
			if FirstMileDriverId != 0 {
				order.PickupRegion = firstRegion
				order.FirstMileDriverId = int(FirstMileDriverId)
				order.FirstMileDriverName = FirstMileDriverName
			}

		}

		//last
		if order.ConsigneeCountry == "Singapore" {
			lastRegion := order.ConsigneePostal
			if len(lastRegion) == 5 {
				lastRegion = "0" + lastRegion
			}

			lastRegion = trimLeadingZeroes(lastRegion[0:2])
			fmt.Println("lastRegion:", lastRegion)
			LastMileDriverId, LastMileDriverName := AssignDriver(lastRegion, order.ConsigneeCountry)
			if LastMileDriverId != 0 {
				order.DeliveryRegion = lastRegion
				order.LastMileDriverId = int(LastMileDriverId)
				order.LastMileDriverName = LastMileDriverName
			}
		}

		//done
		order.AddRegion = "DONE"
		if order.Status == "ORDER_INFO_RECEIVED" && order.FirstMileDriverId != 0 {
			order.Status = "PICK_UP_SCHEDULED"
		}
		if order.Status == "ORDER_RECEIVED_AT_LOCAL_SORTING_CENTER" && order.LastMileDriverId != 0 {
			order.Status = "DELIVERY_SCHEDULED"
		}

		db.DbManager().Save(&order)
	}
}

func trimLeadingZeroes(input string) string {
	inputInt, _ := strconv.Atoi(input)
	return strconv.Itoa(inputInt)
}
