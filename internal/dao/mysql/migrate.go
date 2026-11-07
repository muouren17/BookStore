package mysql

import (
	"bs/internal/model"
	"log"

	"gorm.io/gorm"
)

//数据库建表

func CreateTable(db *gorm.DB) (err error) {
	err = db.AutoMigrate(
		&model.Shelf{},
		&model.Book{},
	)
	if err != nil {
		log.Printf("db.AutoMigrate failed , err:%v\n", err)
	}
	return
}
