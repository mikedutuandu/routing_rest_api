package permission

import (
	"janio-backend/model"
)

func OwnerPermissionPickup(claims *model.JwtCustomClaims,pickup *model.Pickup) bool {

	isAdmin := claims.User.IsAdmin
	if isAdmin == true {
		return true
	}else{
		userId := claims.User.ID
		if int(userId) == pickup.DriverId {
			return  true
		}else{
			return false
		}
	}
}