package utils

import "time"

func GenerateTimeSlots(start, end time.Time, interval time.Duration) []time.Time {
	var slots []time.Time
	for t := start; t.Before(end); t = t.Add(interval) {
		slots = append(slots, t)
	}
	return slots
}

func Ptr[T any](value T) *T {
	return &value
}
