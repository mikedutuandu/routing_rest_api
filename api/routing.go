package api

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"janio-backend/db"
	"janio-backend/helper"
	"janio-backend/model"
	"net/http"
	"strings"
	"time"
)

func UpdateLatLngOrder(c echo.Context) error {
	routingId := c.Param("id")

	url := "http://janio-geocode.herokuapp.com/api/v1/geocode/" + routingId
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]interface{})

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	importId := 0
	m := d["places"].(map[string]interface{})
	for k, v := range m {

		dataOrder := strings.Split(k, "_")
		orderId := dataOrder[0]
		typeAddress := dataOrder[1]

		lat := float64(-1)
		lng :=  float64(-1)
		n := v.(map[string]interface{})
		if n["latitude"] != nil {
			lat = n["latitude"].(float64)
		}
		if n["longitude"] != nil {
			lng = n["longitude"].(float64)
		}

		fmt.Println("lat:", lat)
		fmt.Println("lng:", lng)
		fmt.Println("id:", orderId)
		fmt.Println("typeAddress:", typeAddress)

		order := model.Order{}
		db.DbManager().Where("id = ?", orderId).Find(&order)
		importId = order.ImportId
		if typeAddress == "pickup" {
			order.PickupLat = lat
			order.PickupLng = lng
		} else {
			order.DeliveryLat = lat
			order.DeliveryLng = lng
		}
		if order.PickupLat != 0 && order.PickupLng != 0 && order.DeliveryLat != 0 && order.DeliveryLng != 0 {
			order.AddLatLng = "DONE"
		}
		db.DbManager().Save(&order)

	}
	Import := &model.Import{}
	if importId != 0 {
		db.DbManager().Where("id = ?", importId).Find(Import)
		countNotDone := 0
		db.DbManager().Table("orders").Where("import_id = ? and add_lat_lng = ?",importId,"PENDING").Count(&countNotDone)
		if countNotDone == 0 {
			Import.GeocodeDeliveryStatus = "DONE"
			db.DbManager().Save(Import)
		}
	}
	if Import.GeocodePickupStatus == "DONE" && Import.GeocodeDeliveryStatus == "DONE" {
		Import.StartGeofence = time.Now().Unix()
		helper.GeofenceRegion(Import)
		Import.EndGeofence = time.Now().Unix()

		helper.UpdateDriverForPickups(Import)
		helper.UpdateNumberOrderForEachPickup(Import)

		Import.EndGeocode = time.Now().Unix()

		Import.GeocodeTime = Import.EndGeocode - Import.StartGeocode
		Import.GeofenceTime = Import.EndGeofence - Import.StartGeofence
		Import.UpdatedAt = time.Now()
		Import.Status = "SUCCESS"

		db.DbManager().Save(Import)
	}

	payload := &PayloadSuccess{
		Data: "ok",
	}

	return c.JSON(http.StatusOK, payload)
}

func UpdateLatLngPickup(c echo.Context) error {
	routingId := c.Param("id")

	url := "http://janio-geocode.herokuapp.com/api/v1/geocode/" + routingId
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]interface{})

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	importId := 0
	m := d["places"].(map[string]interface{})
	for k, v := range m {

		pickupId := k

		lat := float64(0)
		lng :=  float64(0)
		n := v.(map[string]interface{})
		if n["latitude"] != nil {
			lat = n["latitude"].(float64)
		}
		if n["longitude"] != nil {
			lng = n["longitude"].(float64)
		}

		fmt.Println("lat:", lat)
		fmt.Println("lng:", lng)
		fmt.Println("id:", pickupId)

		pickup := model.Pickup{}
		db.DbManager().Where("id = ?", pickupId).Find(&pickup)
		importId = pickup.ImportId
		pickup.PickupLat = lat
		pickup.PickupLng = lng
		pickup.AddLatLng = "DONE"
		db.DbManager().Save(&pickup)
	}

	Import := &model.Import{}
	if importId != 0 {
		db.DbManager().Where("id = ?", importId).Find(Import)
		countNotDone := 0
		db.DbManager().Table("pickups").Where("import_id = ? and add_lat_lng = ?",importId,"PENDING").Count(&countNotDone)

		fmt.Println("COUNT_PICK:",countNotDone)
		if countNotDone == 0 {

			fmt.Println("COUNT_PICK_UPDATE:",countNotDone)

			Import.GeocodePickupStatus = "DONE"
			db.DbManager().Save(Import)
		}
	}

	if Import.GeocodePickupStatus == "DONE" && Import.GeocodeDeliveryStatus == "DONE" {
		Import.StartGeofence = time.Now().Unix()
		helper.GeofenceRegion(Import)
		Import.EndGeofence = time.Now().Unix()
		helper.UpdateDriverForPickups(Import)
		helper.UpdateNumberOrderForEachPickup(Import)
		Import.EndGeocode = time.Now().Unix()
		Import.GeocodeTime = Import.EndGeocode - Import.StartGeocode
		Import.GeofenceTime = Import.EndGeofence - Import.StartGeofence
		Import.UpdatedAt = time.Now()
		Import.Status = "SUCCESS"

		db.DbManager().Save(Import)
	}

	payload := &PayloadSuccess{
		Data: "ok",
	}

	return c.JSON(http.StatusOK, payload)
}

func UpdatePositionPickup(c echo.Context) error {

	defer c.Request().Body.Close()

	b, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]interface{})

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	fmt.Print("UpdatePositionPickup:")
	fmt.Print(d)

	driverId := "driver"

	if d["output"] != nil {
		output := d["output"].(map[string]interface{})
		solution := output["solution"].(map[string]interface{})
		dataRouting := solution[driverId].([]interface{})
		for k, v := range dataRouting {
			pickupData := v.(map[string]interface{})
			pickupId := pickupData["location_id"].(string)

			pickup := model.Pickup{}
			db.DbManager().Where("id = ?", pickupId).Find(&pickup)
			if pickup.ID != 0 {
				pickup.ArrivalTime = pickupData["arrival_time"].(string)
				pickup.FinishTime = pickupData["finish_time"].(string)
				pickup.Duration = int32(pickupData["duration"].(float64))
				pickup.SolutionOrder = int32(k)
				db.DbManager().Save(&pickup)
			}
		}
	}

	jobId := c.QueryParam("job_id")
	UpdateData := map[string]interface{}{
		"status": "DONE",
	}
	db.DbManager().Table("optimize_jobs").Where("id = ?", jobId).Updates(UpdateData)

	payload := &PayloadSuccess{
		Data: "ok",
	}

	return c.JSON(http.StatusOK, payload)
}

func UpdatePositionOrder(c echo.Context) error {

	defer c.Request().Body.Close()

	b, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}
	d := make(map[string]interface{})

	err = json.Unmarshal(b, &d)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	driverId := "driver"
	if d["output"] != nil {
		output := d["output"].(map[string]interface{})
		solution := output["solution"].(map[string]interface{})
		dataRouting := solution[driverId].([]interface{})
		for k, v := range dataRouting {
			pickupData := v.(map[string]interface{})
			orderId := pickupData["location_id"].(string)

			order := model.Order{}
			db.DbManager().Where("id = ?", orderId).Find(&order)
			if order.ID != 0 {
				order.ArrivalTime = pickupData["arrival_time"].(string)
				order.FinishTime = pickupData["finish_time"].(string)
				order.Duration = int32(pickupData["duration"].(float64))
				order.SolutionOrder = int32(k)
				db.DbManager().Save(&order)
			}
		}
	}

	jobId := c.QueryParam("job_id")
	UpdateData := map[string]interface{}{
		"status": "DONE",
	}
	db.DbManager().Table("optimize_jobs").Where("id = ?", jobId).Updates(UpdateData)

	payload := &PayloadSuccess{
		Data: "ok",
	}

	return c.JSON(http.StatusOK, payload)
}

func DetailOptimizeJob(c echo.Context) error {
	Id := c.Param("id")

	OptimizeJob := model.OptimizeJob{}

	db.DbManager().Where("id = ?", Id).Find(&OptimizeJob)
	if OptimizeJob.ID == 0 {
		payload := &PayloadError{
			Errors: "Job not found",
		}
		return c.JSON(http.StatusNotFound, payload)
	}

	payload := &PayloadSuccess{
		Data: &OptimizeJob,
	}

	return c.JSON(http.StatusOK, payload)
}

func TestGetRoute(c echo.Context) error {

	orders := []*model.Order{}
	db.DbManager().Where("id IN (?)", []string{"1", "2", "3"}).Find(&orders)
	//
	helper.GetOrderRegionForPickupAndDelivery(orders)

	//pickups := []*model.Pickup{}
	//db.DbManager().Where("id IN (?)",[]string{"1","2","3"} ).Find(&pickups)
	//
	//helper.GetPickupLatLng(pickups)

	payload := &PayloadSuccess{
		Data: "ok",
	}

	return c.JSON(http.StatusOK, payload)
}
