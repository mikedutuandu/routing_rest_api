package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	//_ "github.com/jinzhu/gorm/dialects/sqlite"
	"janio-backend/config"
)

var db *gorm.DB
var err error

func Init() {
	configuration := config.GetConfig()
	connectString := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", configuration.DB_USERNAME, configuration.DB_PASSWORD,configuration.DB_HOST, configuration.DB_NAME)
	db, err = gorm.Open("mysql", connectString)
	//db, err := gorm.Open("sqlite3", "test.db")
	// defer db.Close()
	if err != nil {
		//print(connect_string)
		panic("DB Connection Error")
	}
	//db.AutoMigrate(&model.User{})
	//db.AutoMigrate(&model.Order{})
	//db.AutoMigrate(&model.PostalCode{})
	//db.AutoMigrate(&model.FirstMilePodOrder{})
	//db.AutoMigrate(&model.LastMilePodOrder{})
	//db.AutoMigrate(&model.OptimizeJob{})

}

func DbManager() *gorm.DB {
	return db
}
