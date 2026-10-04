package domain

import (
	"fmt"
	"time"
)

// Date is a calendar day, independent of a clock or timezone.
type Date string

func ParseDate(value string) (Date, error) {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return "", fmt.Errorf("date must use YYYY-MM-DD")
	}
	for i, ch := range value {
		if i == 4 || i == 7 {
			continue
		}
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("date must use YYYY-MM-DD")
		}
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil || t.Year() == 0 || t.Format("2006-01-02") != value {
		return "", fmt.Errorf("invalid calendar date")
	}
	return Date(value), nil
}

func (d Date) String() string { return string(d) }

func (d Date) Time() time.Time {
	t, _ := time.Parse("2006-01-02", string(d))
	return t
}

func ValidateRange(from, to Date) error {
	if _, err := ParseDate(string(from)); err != nil {
		return fmt.Errorf("invalid from date")
	}
	if _, err := ParseDate(string(to)); err != nil {
		return fmt.Errorf("invalid to date")
	}
	if to < from || to.Time().Sub(from.Time()) > 92*24*time.Hour {
		return fmt.Errorf("date range must be 1 to 93 days")
	}
	return nil
}
