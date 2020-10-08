package helper

import (
	"encoding/json"
	"fmt"
	//"janio-backend/config"
	"janio-backend/db"
	"janio-backend/model"
	"regexp"
	"strings"
	"time"
)

func AssignDriver(region string, country string) (uint, string) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()
	user := model.User{}
	db.DbManager().Where("postal_code LIKE ? and country = ?", "%["+region+"]%", country).First(&user)

	if user.ID == 0 {
		return 0, ""
	}

	return user.ID, user.DriverName
}

func GetPostalCodeFromPickupAddress(address string) string {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()
	start := len(address) - 6
	matched, _ := regexp.MatchString("SG$", address)
	if matched == true {
		start = start - 3
	}
	return address[start:]
}

func IsAssignedToDriverForUpdate(postalCodeStr string, currentUser *model.User) bool {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()
	postalCodeArr := strings.Split(postalCodeStr, ",")

	for _, code := range postalCodeArr {
		user := model.User{}
		db.DbManager().Where("id <> ? and postal_code LIKE ?", currentUser.ID, "%["+code+"]%").First(&user)
		if user.ID != 0 {
			return false
		}
	}
	return true
}

func IsAssignedToDriverForCreate(postalCodeStr string) bool {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()
	if postalCodeStr == "" {
		return true
	}
	postalCodeArr := strings.Split(postalCodeStr, ",")

	for _, code := range postalCodeArr {
		user := model.User{}
		db.DbManager().Where("postal_code LIKE ?", "%["+code+"]%").First(&user)
		if user.ID != 0 {
			return false
		}
	}
	return true
}

func BuildStatus(current string, status string, note string) string {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	decode := []map[string]string{}
	json.Unmarshal([]byte(current), &decode)
	item := map[string]string{
		"status": status,
		"note":   note,
		"date":   time.Now().String(),
	}

	newStatus := append(decode, item)
	newStatusEncodeByte, _ := json.Marshal(newStatus)
	newStatusEncodeStr := string(newStatusEncodeByte)

	return newStatusEncodeStr
}

func GetAvatarUrl(u *model.User) string {
	if u.Avatar != "" {
		url := "https://%s.s3.amazonaws.com/%s/%s"
		url = fmt.Sprintf(url, "janio-dev", "avatars", u.Avatar)
		return url
	}
	return ""
}
func GetS3Url(name string, path string) string {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()
	if name != "" {
		url := "https://%s.s3.amazonaws.com/%s/%s"
		url = fmt.Sprintf(url, "janio-dev", path, name)
		return url
	}
	return ""
}

func BuildFileName(fileName string) string {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()
	fileNameArr := strings.Split(fileName, ".")
	ext := fileNameArr[1]
	return time.Now().Format("20060102150405") + "." + ext
}

func ValidateLineOrder(index int, line []string, assignmentType string) (bool, int, map[string]interface{}) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()

	errMsg := make(map[string]interface{})
	canImport := true
	orderId := line[0]
	if orderId == "" {
		errMsg["order_id"] = "Cannot be blank"
		canImport = false
	}
	PickupAddress := line[27]
	if PickupAddress == "" {
		errMsg["pickup_address"] = "Cannot be blank"
		canImport = false
	}
	TrackerStatusCode := line[49]
	if TrackerStatusCode == "" {
		errMsg["tracker_status_code"] = "Cannot be blank"
		canImport = false
	}
	//ORDER_INFO_RECEIVED
	//PICKUP
	//ORIGIN_WAREHOUSE_RECEIVED
	//DELIVERY_IN_PROGRESS
	//DELIVER_FAIL
	//PICKUP_FAIL
	//COMPLETED

	//var strSlice = []string{"ORDER_INFO_RECEIVED", "PICKUP", "ORIGIN_WAREHOUSE_RECEIVED", "DELIVERY_IN_PROGRESS", "DELIVER_FAIL", "PICKUP_FAIL", "COMPLETED"}
	//if ItemExists(strSlice, TrackerStatusCode) == false {
	//	errMsg["tracker_status_code"] = "Have wrong status : " + TrackerStatusCode
	//	canImport = false
	//}

	if assignmentType != "geofencing" {
		ConsigneePostal := line[11]
		if ConsigneePostal == "" || len(ConsigneePostal) < 2 {
			errMsg["consignee_postal"] = "Cannot be blank"
			canImport = false
		}

		PickupPostal := line[32]
		if PickupPostal == "" || len(PickupPostal) < 2 {
			errMsg["pickup_postal"] = "Cannot be blank"
			canImport = false
		}
	}

	ConsigneeAddress := line[5]
	if ConsigneeAddress == "" {
		errMsg["consignee_address"] = "Cannot be blank"
		canImport = false
	}

	if canImport == false {
		errMsg["row"] = index
	}
	return canImport, index, errMsg

}

func IsEmptyRow(line []string) bool {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("ERROR MUST FIX: ", err)
		}
	}()
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
	TrackerStatusCode := line[49]
	TrackingNo := line[51]

	if PickupPostal == "" &&
		PickupAddress == "" &&
		PickupCity == "" &&
		PickupContactName == "" &&
		PickupContactNumber == "" &&
		PickupCountry == "" &&
		PickupProvince == "" &&
		PickupState == "" &&
		ConsigneeAddress == "" &&
		ConsigneeCity == "" &&
		ConsigneeCountry == "" &&
		ConsigneeEmail == "" &&
		ConsigneeName == "" &&
		ConsigneeNumber == "" &&
		ConsigneePostal == "" &&
		ConsigneeProvince == "" &&
		ConsigneeState == "" &&
		DeliveryNote == "" &&
		TrackerStatusCode == "" &&
		TrackingNo == "" {
		return true
	}
	return false
}

func UpdateStatusOrdersForPickup(listOrdersStatus string) string {
	allPickupFail := true
	allPickupComplete := true

	//b :=`[{"id":1,"status":1,"status_note":"haha"},{"id":2,"status":2,"status_note":"haha"}]`
	d := []map[string]interface{}{}

	err := json.Unmarshal([]byte(listOrdersStatus), &d)
	if err != nil {
		fmt.Println("ERROR MUST FIX: ", err)
	}
	for _, item := range d {
		status := item["status"].(string)
		exOrderId := item["id"].(string)
		statusNote := item["status_note"].(string)
		if ItemExists([]string{"ORDER_PICKED_UP", "PICKUP_FAILED"}, status) == true {
			//find
			order := model.Order{}
			db.DbManager().Where("tracking_no = ?", exOrderId).Find(&order)
			//update
			statusNote := BuildStatus(order.StatusNote, status, statusNote)
			order.Status = status
			order.StatusNote = statusNote
			db.DbManager().Save(&order)

			//check status for pickup
			if status == "PICKUP_FAILED" {
				allPickupComplete = false
			}
			if status == "ORDER_PICKED_UP" {
				allPickupFail = false
			}
		}
	}

	//check status for pickup
	if allPickupComplete == true {
		return "COMPLETED"
	} else if allPickupFail == true {
		return "FAILED"
	} else {
		return "COMPLETED"
	}

}

//func PreparePostalCode(postalCodeStr string) string {
//	defer func() {
//		if err := recover(); err != nil{
//			fmt.Println("ERROR MUST FIX: ",err)
//		}
//	}()
//	postalCodeArr := strings.Split(postalCodeStr, ",")
//	postalCodeArrNew := []string{}
//
//
//	for _, code := range postalCodeArr {
//		postalCodeArrNew = append(postalCodeArrNew,"["+code+"]")
//	}
//	return 	strings.Join(postalCodeArr,",")
//}
