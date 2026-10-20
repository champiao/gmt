package models

import "time"

type SchemaMigration struct {
	Name      string `gorm:"primaryKey"`
	AppliedAt time.Time
}

type SchemaSeed struct {
	Name      string `gorm:"primaryKey"`
	AppliedAt time.Time
}
