package handler

import (
	"time"
)

func parseDueDate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		t2, err2 := time.Parse("2006-01-02", *s)
		if err2 != nil {
			return nil
		}
		return &t2
	}
	return &t
}
