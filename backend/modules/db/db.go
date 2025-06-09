package db

import (
	"spacenode/libs/models"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB(path string) {
	var err error
	db, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		logrus.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&models.AppNode{})
	logrus.Infoln("Database connection established")
}

func DB() *gorm.DB {
	if db == nil {
		logrus.Fatal("Database not initialized. Call InitDB first.")
	}
	return db
}
