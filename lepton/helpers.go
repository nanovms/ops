package lepton

import (
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

// time stolen from https://raw.githubusercontent.com/dustin/go-humanize/master/times.go

// Seconds-based time units
const (
	Day      = 24 * time.Hour
	Week     = 7 * Day
	Month    = 30 * Day
	Year     = 12 * Month
	LongTime = 37 * Year

	KB = 1000
	MB = 1000 * KB
	GB = 1000 * MB
	TB = 1000 * GB
	PB = 1000 * TB

	KiB = 1024
	MiB = 1024 * KiB
	GiB = 1024 * MiB
	TiB = 1024 * GiB
	PiB = 1024 * TiB
)

// Time2Human formats a time into a relative string.
// Time(someT) -> "3 weeks ago"
func Time2Human(then time.Time) string {
	return RelTime(then, time.Now(), "ago", "from now")
}

// A RelTimeMagnitude struct contains a relative time point at which
// the relative format of time will switch to a new format string.  A
// slice of these in ascending order by their "D" field is passed to
// CustomRelTime to format durations.
//
// The Format field is a string that may contain a "%s" which will be
// replaced with the appropriate signed label (e.g. "ago" or "from
// now") and a "%d" that will be replaced by the quantity.
//
// The DivBy field is the amount of time the time difference must be
// divided by in order to display correctly.
//
// e.g. if D is 2*time.Minute and you want to display "%d minutes %s"
// DivBy should be time.Minute so whatever the duration is will be
// expressed in minutes.
type RelTimeMagnitude struct {
	D      time.Duration
	Format string
	DivBy  time.Duration
}

type unitMap map[byte]int64

var defaultMagnitudes = []RelTimeMagnitude{
	{time.Second, "now", time.Second},
	{2 * time.Second, "1 second %s", 1},
	{time.Minute, "%d seconds %s", time.Second},
	{2 * time.Minute, "1 minute %s", 1},
	{time.Hour, "%d minutes %s", time.Minute},
	{2 * time.Hour, "1 hour %s", 1},
	{Day, "%d hours %s", time.Hour},
	{2 * Day, "1 day %s", 1},
	{Week, "%d days %s", Day},
	{2 * Week, "1 week %s", 1},
	{Month, "%d weeks %s", Week},
	{2 * Month, "1 month %s", 1},
	{Year, "%d months %s", Month},
	{18 * Month, "1 year %s", 1},
	{2 * Year, "2 years %s", 1},
	{LongTime, "%d years %s", Year},
	{math.MaxInt64, "a long while %s", 1},
}

var (
	decimalMap   = unitMap{'k': KB, 'm': MB, 'g': GB, 't': TB, 'p': PB}
	binaryMap    = unitMap{'k': KiB, 'm': MiB, 'g': GiB, 't': TiB, 'p': PiB}
	decimapAbbrs = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	binaryAbbrs  = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
)

// RelTime formats a time into a relative string.
//
// It takes two times and two labels.  In addition to the generic time
// delta string (e.g. 5 minutes), the labels are used applied so that
// the label corresponding to the smaller time is applied.
//
// RelTime(timeInPast, timeInFuture, "earlier", "later") -> "3 weeks
// earlier"
func RelTime(a, b time.Time, albl, blbl string) string {
	return CustomRelTime(a, b, albl, blbl, defaultMagnitudes)
}

// CustomRelTime formats a time into a relative string.
//
// It takes two times two labels and a table of relative time formats.
// In addition to the generic time delta string (e.g. 5 minutes), the
// labels are used applied so that the label corresponding to the
// smaller time is applied.
func CustomRelTime(a, b time.Time, albl, blbl string, magnitudes []RelTimeMagnitude) string {
	lbl := albl
	diff := b.Sub(a)

	if a.After(b) {
		lbl = blbl
		diff = a.Sub(b)
	}

	n := sort.Search(len(magnitudes), func(i int) bool {
		return magnitudes[i].D > diff
	})

	if n >= len(magnitudes) {
		n = len(magnitudes) - 1
	}
	mag := magnitudes[n]
	args := []interface{}{}
	escaped := false
	for _, ch := range mag.Format {
		if escaped {
			switch ch {
			case 's':
				args = append(args, lbl)
			case 'd':
				args = append(args, diff/mag.DivBy)
			}
			escaped = false
		} else {
			escaped = ch == '%'
		}
	}
	return fmt.Sprintf(mag.Format, args...)
}

// Bytes2Human parses bytes to other byte units as MegaByte and Gigabyte.
func Bytes2Human(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

// RAMInBytes parses a human-readable string representing an amount of RAM
// in bytes, kibibytes, mebibytes, gibibytes, or tebibytes and
// returns the number of bytes, or -1 if the string is unparseable.
// Units are case-insensitive, and the 'b' suffix is optional.
func RAMInBytes(size string) (int64, error) {
	return parseSize(size, binaryMap)
}

// CCidr converts network mask to CIDR size.
func CCidr(netmask string) int {
	ip := net.ParseIP(netmask)
	sz, _ := net.IPMask(ip.To4()).Size()
	return sz
}

// parseSize parses the human-readable size string into the amount it represents.
func parseSize(sizeStr string, uMap unitMap) (int64, error) {
	// TODO: rewrite to use strings.Cut if there's a space
	// once Go < 1.18 is deprecated.
	sep := strings.LastIndexAny(sizeStr, "01234567890. ")
	if sep == -1 {
		// There should be at least a digit.
		return -1, fmt.Errorf("invalid size: '%s'", sizeStr)
	}
	var num, sfx string
	if sizeStr[sep] != ' ' {
		num = sizeStr[:sep+1]
		sfx = sizeStr[sep+1:]
	} else {
		// Omit the space separator.
		num = sizeStr[:sep]
		sfx = sizeStr[sep+1:]
	}

	size, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return -1, err
	}
	// Backward compatibility: reject negative sizes.
	if size < 0 {
		return -1, fmt.Errorf("invalid size: '%s'", sizeStr)
	}

	if len(sfx) == 0 {
		return int64(size), nil
	}

	// Process the suffix.

	if len(sfx) > 3 { // Too long.
		goto badSuffix
	}
	sfx = strings.ToLower(sfx)
	// Trivial case: b suffix.
	if sfx[0] == 'b' {
		if len(sfx) > 1 { // no extra characters allowed after b.
			goto badSuffix
		}
		return int64(size), nil
	}
	// A suffix from the map.
	if mul, ok := uMap[sfx[0]]; ok {
		size *= float64(mul)
	} else {
		goto badSuffix
	}

	// The suffix may have extra "b" or "ib" (e.g. KiB or MB).
	switch {
	case len(sfx) == 2 && sfx[1] != 'b':
		goto badSuffix
	case len(sfx) == 3 && sfx[1:] != "ib":
		goto badSuffix
	}

	return int64(size), nil

badSuffix:
	return -1, fmt.Errorf("invalid suffix: '%s'", sfx)
}
