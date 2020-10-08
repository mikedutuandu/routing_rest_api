package services

import (
	"janio-backend/constant"
	"janio-backend/db"
	"time"
)

var timzone string = "Asia/Singapore"
var loc, _ = time.LoadLocation(timzone)
var today = time.Now().In(loc).Format(constant.DATE_LAYOUT_ISO)

const BATCH_UPDATE int = 10000

func ReschedulePickups() {
	go updatePickups()
	go updateOrders()
}

func updatePickups() {
	for {
		query := db.DbManager().Table("pickups")
		query = query.Where("reschedule_date = ?", today)
		query = query.Where("status = (?) ", constant.STATUS_PICKUP_RESCHEDULED)
		query = query.Limit(BATCH_UPDATE)

		data := map[string]interface{}{}
		data["status"] = constant.STATUS_PICK_UP_SCHEDULED
		data["reschedule_date"] = nil

		rowsAffected := query.Updates(data).RowsAffected
		// fmt.Println(strconv.Itoa(int(rowsAffected)))

		if rowsAffected == 0 {
			break
		}
	}
}

func updateOrders() {
	for {
		query := db.DbManager().Table("orders")
		query = query.Where("reschedule_date = ?", today)
		query = query.Where("status = (?) ", "DELIVERY_RESCHEDULED")
		query = query.Limit(BATCH_UPDATE)

		data := map[string]interface{}{}
		data["status"] = "DELIVERY_SCHEDULED"
		data["reschedule_date"] = nil

		rowsAffected := query.Updates(data).RowsAffected
		// fmt.Println(strconv.Itoa(int(rowsAffected)))

		if rowsAffected == 0 {
			break
		}
	}
}
