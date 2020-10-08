package validator

import (
	"encoding/json"
	"janio-backend/db"
	"janio-backend/helper"
	"janio-backend/model"
	"net/url"
	"fmt"
)

func OptimizePickupValidator(data map[string]float64)  (err url.Values)  {
	errs :=  url.Values{}
	if data["lat"] == 0 {
		errs.Add("lat", "The lat is required!")
	}
	if data["lng"] == 0 {
		errs.Add("lng", "The lng is required!")
	}

	return errs
}


func ValidateCanUpdatePickup(isAdmin bool,status string)bool{
	defer func() {
		if err := recover(); err != nil{
			fmt.Println("ERROR MUST FIX: ",err)
		}
	}()
	if isAdmin == true{
		return true
	}
	if status == "COMPLETED"{
		return false
	}
	return true
}
func ValidateOrderStatusListForPickup(pickupId int,listOrdersStatus string)(bool,int){
	defer func() {
		if err := recover(); err != nil{
			fmt.Println("ERROR MUST FIX: ",err)
		}
	}()
	d := []map[string]interface{}{}

	err := json.Unmarshal([]byte(listOrdersStatus), &d)
	if err != nil{
		fmt.Println("ERROR MUST FIX: ",err)
	}
	countOrder := 0
	db.DbManager().Model(&model.Order{}).Where("pickup_id = ?", pickupId).Count(&countOrder)
	if countOrder != len(d){
		return false,0
	}

	numberOrderPickup := 0
	for _,item := range d {
		exOrderId := item["id"].(string)
		order := model.Order{}
		db.DbManager().Where("tracking_no = ?",exOrderId ).Find(&order)
		if order.PickupId != pickupId {
			return false,0
		}
		status := item["status"].(string)
		if helper.ItemExists([]string{"ORDER_PICKED_UP", "PICKUP_FAILED"}, status) == false {
			return false,0
		}
		if status == "ORDER_PICKED_UP"{
			numberOrderPickup = numberOrderPickup +1
		}
	}
	return true,numberOrderPickup
}