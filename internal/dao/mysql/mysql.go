package mysql

import (
	"bs/settings"
	"log"
	"strconv"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 数据库初始化
func Init(cfg *settings.MySQLConfig) (db *gorm.DB, err error) {
	var sb strings.Builder
	sb.WriteString(cfg.User)
	sb.WriteString(":")
	sb.WriteString(cfg.Password)
	sb.WriteString("@tcp(")
	sb.WriteString(cfg.Host)
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(cfg.Port))
	sb.WriteString(")/")
	sb.WriteString(cfg.DbName)
	sb.WriteString("?charset=utf8mb4&parseTime=True&loc=Local")
	dsn := sb.String()
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("gorm.Open failed, err:%v\n", err)
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("db.DB failed, err:%v\n", err)
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	return db, nil
}
