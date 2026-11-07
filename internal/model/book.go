package model

import "time"

type Book struct {
	ID        int64 `gorm:"primaryKey"`
	Author    string
	Title     string
	ShelfID   int64 `gorm:"not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
