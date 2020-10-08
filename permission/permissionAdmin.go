package permission

import (
	"janio-backend/model"
)

func AdminPermission(claims *model.JwtCustomClaims) bool {

	isAdmin := claims.User.IsAdmin
	return isAdmin
}