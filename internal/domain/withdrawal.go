package domain

import "time"

type Withdrawal struct {
	OrderNumber string
	Sum         Kopecks
	ProcessedAt time.Time
}
