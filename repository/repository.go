package repository

import (
	"gorm.io/gorm"
)

func RetrieveBy(db *gorm.DB, foundRows any, valueName string, value any) *gorm.DB {
	return db.Where(valueName+" = ?", value).Find(foundRows)
}

func RetrieveByID(db *gorm.DB, foundRow any, id int) *gorm.DB {
	return db.Where("id = ?", id).First(foundRow)
}

func UpdateBy(db *gorm.DB, model any, updatedValue any, valueName string, value any) *gorm.DB {
	return db.Model(model).Where(valueName+" = ?", value).Updates(updatedValue)
}
