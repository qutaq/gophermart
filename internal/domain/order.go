package domain

import "time"

type Order struct {
	Number     string
	UserID     int64
	Status     string
	Accrual    *Kopecks
	UploadedAt time.Time
}
