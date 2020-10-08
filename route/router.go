package route

import (
	"github.com/labstack/gommon/log"
	"janio-backend/api"
	"janio-backend/model"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func Init() *echo.Echo {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))

	e.Static("/static", "assets")


	// Login route
	e.POST("/api/auth/token", api.Login)

	// Create user
	e.POST("/api/auth/create", api.CreateAdmin)

	e.GET("/api/update-lat-lng-order/:id", api.UpdateLatLngOrder)
	e.GET("/api/update-lat-lng-pickup/:id", api.UpdateLatLngPickup)

	e.POST("/api/update-position-pickup", api.UpdatePositionPickup)
	e.POST("/api/update-position-order", api.UpdatePositionOrder)


	e.GET("/api/test-update-lat-lng", api.TestGetRoute)
	// Restricted group==================================================
	r := e.Group("/api/")

	// Configure middleware with the custom claims type
	config := middleware.JWTConfig{
		Claims:     &model.JwtCustomClaims{},
		SigningKey: []byte("secret"),
		AuthScheme: "JWT",
	}
	r.Use(middleware.JWTWithConfig(config))


	r.GET("", api.Restricted)

	r.POST("drivers", api.CreateDriver)
	r.PUT("drivers/:id", api.UpdateDriver)
	r.GET("drivers", api.ListDriver)


	r.POST("orders/upload", api.UploadOrder)
	r.GET("orders/upload/:id", api.DetailUploadOrder)
	r.GET("orders/upload/list", api.ListImport)

	r.PUT("orders/:id", api.UpdateOrder)
	r.PUT("orders/:id/admin", api.AdminUpdateOrder)
	r.POST("orders", api.AdminCreateOrder)
	r.GET("orders/:id", api.DetailOrder)
	r.DELETE("orders/:id/delete-first-mile-signature", api.DeleteFirstMileCustomerSignatureOrder)//
	r.DELETE("orders/:id/delete-last-mile-signature", api.DeleteLastMileCustomerSignatureOrder)//
	r.POST("orders/:id/upload-first-mile-signature", api.UploadFirstMileCustomerSignatureOrder)//
	r.POST("orders/:id/upload-last-mile-signature", api.UploadLastMileCustomerSignatureOrder)//
	r.GET("orders", api.ListOrder)

	r.POST("orders/optimize", api.OptimizeDelivery)

	r.POST("orders/:id/pod-first-mile", api.UploadFirstMilePODOrder)//1
	r.POST("orders/:id/pod-last-mile", api.UploadLastMilePODOrder)//2
	r.POST("orders/:ex_id/scan-to-warehouse", api.ScanToWarehouse)
	r.POST("orders/:ex_id/scan-to-delivery", api.ScanToDelivery)

	r.POST("orders/bulk", api.UpdateBulkOrder)

	r.DELETE("pod-first-mile/:id", api.DeleteFirstMilePODOrder)//3
	r.DELETE("pod-last-mile/:id", api.DeleteLastMilePODOrder)//4
	r.POST("pickups/optimize", api.OptimizePickup)


	r.GET("pickups/:id", api.DetailPickup)
	r.PUT("pickups/:id", api.UpdatePickup)
	r.GET("pickups", api.ListPickup)

	r.POST("pickups/:id/upload-pod", api.UploadPODPickup)//
	r.DELETE("pod-pickup/:id", api.DeletePODPickup)//


	r.GET("postal-code", api.ListPostalCode)

	r.GET("detail-optimize-job/:id", api.DetailOptimizeJob)

	r.GET("reset-orders", api.ResetOrder)




	return e
}
