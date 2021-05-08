package cmd

import (
	"regexp"
	"strconv"
	"time"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/log"
)

// SubtractTimeNotation subtracts time notation timestamp from date passed by argument
func SubtractTimeNotation(date time.Time, notation string) (newDate time.Time, err error) {
	r := regexp.MustCompile(`(\d+)(d|w|m|y){1}`)
	groups := r.FindStringSubmatch(notation)
	if len(groups) == 0 {
		return newDate, errors.New("invalid time notation")
	}

	n, err := strconv.Atoi(groups[1])
	if err != nil {
		log.Error(err.Error())
		return newDate, err
	}

	switch groups[2] {
	case "d":
		return date.AddDate(0, 0, n*-1), nil
	case "w":
		return date.AddDate(0, 0, n*7*-1), nil
	case "m":
		return date.AddDate(0, n*-1, 0), nil
	case "y":
		return date.AddDate(n*-1, 0, 0), nil
	default:
		return date, nil
	}
}
