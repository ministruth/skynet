package db

import "time"

type Track struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Users struct {
	ID       int32  `gorm:"primaryKey;not null"`
	Username string `gorm:"uniqueIndex;type:varchar(32);not null"`
	Password string `gorm:"type:char(32);not null"`
	Avatar   []byte `gorm:"type:bytes;not null"`
	Track    Track  `gorm:"embedded"`
}
