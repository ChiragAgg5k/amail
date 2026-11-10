package mail

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	shortestCode = 4
	longestCode  = 8
	earliestYear = 1900
	latestYear   = 2099
	// joiners link digit groups into dates, times, decimals and phone numbers, which are not codes.
	joiners = "-/.:,"
)

var (
	digitRun = regexp.MustCompile(`[0-9]+`)
	codeWord = regexp.MustCompile(`(?i)code|\botp\b|verification`)
)

// OneTimeCode finds the one-time code in a message's subject and text.
//
// A candidate is a run of 4 to 8 ASCII digits that touches no letter or digit and is not joined to
// another digit by - / . : or , (so parts of dates, times, amounts and phone numbers are skipped).
// The first candidate on a line that mentions a code, OTP, passcode or verification wins; failing
// that, the first candidate anywhere. A 4-digit run from 1900 to 2099 reads as a year: it is only
// taken on a line that mentions a code and has no other candidate, and never in the fallback.
func OneTimeCode(text string) (string, bool) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if !codeWord.MatchString(line) {
			continue
		}
		var year string
		for _, candidate := range candidates(line) {
			if !yearLike(candidate) {
				return candidate, true
			}
			if year == "" {
				year = candidate
			}
		}
		if year != "" {
			return year, true
		}
	}
	for _, line := range lines {
		for _, candidate := range candidates(line) {
			if !yearLike(candidate) {
				return candidate, true
			}
		}
	}
	return "", false
}

func candidates(line string) []string {
	var found []string
	for _, span := range digitRun.FindAllStringIndex(line, -1) {
		start, end := span[0], span[1]
		if length := end - start; length < shortestCode || length > longestCode {
			continue
		}
		if bounded(line, start, end) {
			found = append(found, line[start:end])
		}
	}
	return found
}

// bounded reports whether the digits at line[start:end] stand alone rather than inside a word or a longer number.
func bounded(line string, start, end int) bool {
	if start > 0 {
		before, size := utf8.DecodeLastRuneInString(line[:start])
		if unicode.IsLetter(before) || unicode.IsDigit(before) {
			return false
		}
		if strings.ContainsRune(joiners, before) && start-size > 0 {
			if previous, _ := utf8.DecodeLastRuneInString(line[:start-size]); unicode.IsDigit(previous) {
				return false
			}
		}
	}
	if end < len(line) {
		after, size := utf8.DecodeRuneInString(line[end:])
		if unicode.IsLetter(after) || unicode.IsDigit(after) {
			return false
		}
		if strings.ContainsRune(joiners, after) && end+size < len(line) {
			if next, _ := utf8.DecodeRuneInString(line[end+size:]); unicode.IsDigit(next) {
				return false
			}
		}
	}
	return true
}

func yearLike(candidate string) bool {
	if len(candidate) != shortestCode {
		return false
	}
	year, err := strconv.Atoi(candidate)
	return err == nil && year >= earliestYear && year <= latestYear
}
