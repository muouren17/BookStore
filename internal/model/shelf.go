package model

import "time"

type Shelf struct {
	ID        int64 `gorm:"primaryKey"`
	Theme     string
	Size      int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
