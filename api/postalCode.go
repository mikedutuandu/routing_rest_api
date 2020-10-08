package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)



func ListPostalCode(c echo.Context) error {
	//status := c.QueryParam("status")
	//db := db.DbManager()
	//postalCodes := []*model.PostalCode{}
	//count := 0
	//db.Find(&postalCodes)
	//db.Table("postal_codes").Count(&count)
	//
	//data := make(map[string]interface{})
	//codes := []string{}
	//for _, postalCode := range postalCodes{
	//	userId,_ := helper.AssignDriver(postalCode.PostalCode)
	//	if status == "unassigned" {
	//		if int(userId) == 0 {
	//			codes = append(codes, postalCode.PostalCode)
	//			data[postalCode.CountryCode] = codes
	//		}
	//	}else{
	//		if int(userId) != 0 {
	//			codes = append(codes, postalCode.PostalCode)
	//			data[postalCode.CountryCode] = codes
	//		}
	//	}
	//}

	payload := &PayloadSuccess{
		Data: "Not use this ANYMORE",
		Meta: struct {
			TotalRecord int  `json:"total_record"`
		}{0},
	}
	return c.JSON(http.StatusOK,payload)
}


