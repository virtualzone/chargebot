package main

import "time"

type Time interface {
	UTCNow() time.Time
}

type RealTime struct{}

func (RealTime) UTCNow() time.Time {
	return time.Now().UTC()
}
