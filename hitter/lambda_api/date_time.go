package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

func getHTTPDateString(dateStr string) (string, error) {
	// Must be displayed in GMT time zone
	t, err := parseToGMT(dateStr)
	if err != nil {
		return "", err
	}

	return getHTTPFormattedDateString(t), nil
}

func getFileNameDateString(dateStr string) (string, error) {
	t, err := parseToJST(dateStr)
	if err != nil {
		return "", err
	}

	return getFileNameFormattedDateString(t), nil
}

func getDisplayDateString(dateStr string, limitSeconds string) (string, error) {
	t, err := parseToJST(dateStr)
	if err != nil {
		return "", err
	}

	limit, err := strconv.Atoi(strings.TrimSpace(limitSeconds))
	if err == nil {
		t = t.Add(time.Duration(limit) * time.Second)
	}

	return getFormattedDateString(t), nil
}

func parseToJST(dateStr string) (time.Time, error) {
	var result time.Time

	// Set the locale to JST.
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return result, err
	}

	// Parsing in JST locales
	t, err := dateparse.ParseAny(dateStr)
	if err != nil {
		return result, err
	}
	result = t.In(jst)

	return result, err
}

func parseToGMT(dateStr string) (time.Time, error) {
	var result time.Time

	// Set the locale to GMT.
	gmt, err := time.LoadLocation("GMT")
	if err != nil {
		return result, err
	}

	// Parsing in JST locales
	t, err := dateparse.ParseAny(dateStr)
	if err != nil {
		return result, err
	}
	result = t.In(gmt)

	return result, err
}

func getFormattedDateString(dateTime time.Time) string {
	return dateTime.Format("2006/01/02 Mon 15:04:05 MST")
}

func getFileNameFormattedDateString(dateTime time.Time) string {
	return dateTime.Format("20060102_Mon_150405_MST")
}

// Date: Sat, 23 Dec 2019 06:53:29 GMT
func getHTTPFormattedDateString(dateTime time.Time) string {
	return dateTime.Format("Mon, 02 Jan 2006 15:04:05 MST")
}
