package emailverifier

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	// Standard Errors
	ErrTimeout           = "The connection to the mail server has timed out"
	ErrNoSuchHost        = "Mail server does not exist"
	ErrServerUnavailable = "Mail server is unavailable"
	ErrBlocked           = "Blocked by mail server"

	// RCPT Errors
	ErrTryAgainLater           = "Try again later"
	ErrFullInbox               = "Recipient out of disk space"
	ErrTooManyRCPT             = "Too many recipients"
	ErrNoRelay                 = "Not an open relay"
	ErrMailboxBusy             = "Mailbox busy"
	ErrExceededMessagingLimits = "Messaging limits have been exceeded"
	ErrNotAllowed              = "Not Allowed"
	ErrNeedMAILBeforeRCPT      = "Need MAIL before RCPT"
	ErrRCPTHasMoved            = "Recipient has moved"
	ErrMailboxNotFound         = "Mailbox not found"
)

// LookupError is an MX dns records lookup error
type LookupError struct {
	Message string `json:"message" xml:"message"`
	Details string `json:"details" xml:"details"`
}

// newLookupError creates a new LookupError reference and returns it
func newLookupError(message, details string) *LookupError {
	return &LookupError{message, details}
}

func (e *LookupError) Error() string {
	return fmt.Sprintf("%s : %s", e.Message, e.Details)
}

// ParseSMTPError receives an MX Servers response message
// and generates the corresponding MX error
func ParseSMTPError(err error) *LookupError {
	errStr := err.Error()

	// Verify the length of the error before reading nil indexes
	if len(errStr) < 3 {
		return parseBasicErr(err)
	}

	// Strips out the status code string and converts to an integer for parsing
	status, convErr := strconv.Atoi(string([]rune(errStr)[0:3]))
	if convErr != nil {
		return parseBasicErr(err)
	}

	// If the status code is above 400 there was an error and we should return it
	if status > 400 {
		if status < 500 {
			if insContains(errStr,
				"greylist",
				"greylisted") {
				return newLookupError(ErrTryAgainLater, errStr)
			}

			switch status {
			case 421:
				return newLookupError(ErrTryAgainLater, errStr)
			case 450:
				return newLookupError(ErrMailboxBusy, errStr)
			case 451:
				return newLookupError(ErrExceededMessagingLimits, errStr)
			case 452:
				if insContains(errStr,
					"full",
					"space",
					"over quota",
					"insufficient",
				) {
					return newLookupError(ErrFullInbox, errStr)
				}
				return newLookupError(ErrTooManyRCPT, errStr)
			default:
				return parseBasicErr(err)
			}
		}

		if insContains(errStr,
			"undeliverable",
			"does not exist",
			"may not exist",
			"user unknown",
			"user not found",
			"invalid address",
			"recipient invalid",
			"recipient rejected",
			"address rejected",
			"no mailbox",
			"no mail-enabled") {
			return newLookupError(ErrMailboxNotFound, errStr) // These errors indicate the address doesn't exist, not a server problem
		}

		switch status {
		case 503:
			return newLookupError(ErrNeedMAILBeforeRCPT, errStr)
		case 550: // 550 is Mailbox Unavailable - usually undeliverable, ref: https://blog.mailtrap.io/550-5-1-1-rejected-fix/
			if insContains(errStr,
				"spamhaus",
				"proofpoint",
				"cloudmark",
				"banned",
				"blacklisted",
				"blocked",
				"block list",
				"denied") {
				return newLookupError(ErrBlocked, errStr)
			}
			return newLookupError(ErrMailboxNotFound, errStr)
		case 551:
			return newLookupError(ErrRCPTHasMoved, errStr)
		case 552:
			return newLookupError(ErrFullInbox, errStr)
		case 553:
			return newLookupError(ErrNoRelay, errStr)
		case 554:
			if insContains(errStr, "relay access denied") {
				return newLookupError(ErrNoRelay, errStr)
			}
			return newLookupError(ErrNotAllowed, errStr)
		default:
			return parseBasicErr(err)
		}
	}
	return nil
}

// parseBasicErr parses a basic MX record response and returns
// a more understandable LookupError
func parseBasicErr(err error) *LookupError {
	errStr := err.Error()

	// Return a more understandable error
	switch {
	case errors.Is(err, io.EOF):
		return newLookupError(ErrServerUnavailable, errStr)
	case insContains(errStr,
		"spamhaus",
		"proofpoint",
		"cloudmark",
		"banned",
		"blocked",
		"denied"):
		return newLookupError(ErrBlocked, errStr)
	case insContains(errStr, "timeout"):
		return newLookupError(ErrTimeout, errStr)
	case insContains(errStr, "no such host"):
		return newLookupError(ErrNoSuchHost, errStr)
	case insContains(errStr,
		"unavailable",
		"connection reset"):
		return newLookupError(ErrServerUnavailable, errStr)
	default:
		return newLookupError(errStr, errStr)
	}
}

// insContains returns true if any of the substrings
// are found in the passed string. This method of checking
// contains is case insensitive
func insContains(str string, subStrs ...string) bool {
	for _, subStr := range subStrs {
		if strings.Contains(strings.ToLower(str),
			strings.ToLower(subStr)) {
			return true
		}
	}
	return false
}
