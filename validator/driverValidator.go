package validator

import (
	"janio-backend/db"
	"janio-backend/helper"
	"janio-backend/model"
	"net/url"
)

func CreateDriverValidator(driver *model.User)  (err url.Values)  {
	 errs :=  url.Values{}
	if driver.DriverLicense == "" {
		errs.Add("driver_license", "The driver_license is required!")
	}
	if driver.DriverName == "" {
		errs.Add("driver_name", "The driver_name is required!")
	}
	if driver.LicensePlate == "" {
		errs.Add("license_plate", "The license_plate is required!")
	}

	if driver.Phone == "" {
		errs.Add("phone", "The phone is required!")
	}

	if driver.ShiftPickupStart == "" {
		errs.Add("shift_pickup_start", "The shift_pickup_start is required!")
	}

	if driver.ShiftPickupEnd == "" {
		errs.Add("shift_pickup_end", "The shift_pickup_end is required!")
	}

	if driver.ShiftDeliveryStart == "" {
		errs.Add("shift_delivery_start", "The shift_delivery_start is required!")
	}

	if driver.ShiftDeliveryEnd == "" {
		errs.Add("shift_delivery_end", "The shift_delivery_end is required!")
	}


	//check each postal not belong more than one driver
	isAssigned := helper.IsAssignedToDriverForCreate(driver.PostalCode)
	if isAssigned == false {
		errs.Add("postal_code", "The postal_code is assigned, please check again!")
	}

	user := model.User{}
	db.DbManager().Where("driver_license = ?", driver.DriverLicense).First(&user)
	if user.ID != 0 {
		errs.Add("driver_license", "The driver_license must be unique!")
	}

	return errs
}


func UpdateDriverValidator(driver *model.User)  (err url.Values)  {
	errs :=  url.Values{}
	if driver.DriverLicense == "" {
		errs.Add("driver_license", "The driver_license is required!")
	}
	if driver.DriverName == "" {
		errs.Add("driver_name", "The driver_name is required!")
	}
	if driver.LicensePlate == "" {
		errs.Add("license_plate", "The license_plate is required!")
	}

	if driver.Phone == "" {
		errs.Add("phone", "The phone is required!")
	}

	if driver.ShiftPickupStart == "" {
		errs.Add("shift_pickup_start", "The shift_pickup_start is required!")
	}

	if driver.ShiftPickupEnd == "" {
		errs.Add("shift_pickup_end", "The shift_pickup_end is required!")
	}

	if driver.ShiftDeliveryStart == "" {
		errs.Add("shift_delivery_start", "The shift_delivery_start is required!")
	}

	if driver.ShiftDeliveryEnd == "" {
		errs.Add("shift_delivery_end", "The shift_delivery_end is required!")
	}



	//check each postal not belong more than one driver
	isAssigned := helper.IsAssignedToDriverForUpdate(driver.PostalCode,driver)
	if isAssigned == false {
		errs.Add("postal_code", "The postal_code is assigned, please check again!")
	}

	user := model.User{}
	db.DbManager().Where("id <> ? and driver_license = ?", driver.ID, driver.DriverLicense).First(&user)
	if user.ID != 0 {
		errs.Add("driver_license", "The driver_license must be unique!")
	}

	return errs
}

