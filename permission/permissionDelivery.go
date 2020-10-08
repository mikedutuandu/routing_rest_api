package permission

import (
	"janio-backend/model"
)

func OwnerPermissionDelivery(claims *model.JwtCustomClaims, order *model.Order) bool {

	isAdmin := claims.User.IsAdmin
	if isAdmin == true {
		return true
	}
	userId := claims.User.ID
	if int(userId) == order.LastMileDriverId {
		return true
	} else {
		return false
	}
}
