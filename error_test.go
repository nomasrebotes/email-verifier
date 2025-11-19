package emailverifier

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse550RCPTError(t *testing.T) {
	err := errors.New("550 This mailbox does not exist")
	le := ParseSMTPError(err)
	assert.Equal(t, ErrMailboxNotFound, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParse550BlockedRCPTError(t *testing.T) {
	err := errors.New("550 spamhaus")
	le := ParseSMTPError(err)
	assert.Equal(t, ErrBlocked, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseConnectMailExchangerError(t *testing.T) {
	err := errors.New("Timeout connecting to mail-exchanger")
	le := ParseSMTPError(err)
	assert.Equal(t, ErrTimeout, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseNoMxRecordsFoundError(t *testing.T) {
	errStr := "No MX records found"
	err := errors.New(errStr)
	le := ParseSMTPError(err)
	assert.Equal(t, &LookupError{Details: errStr, Message: errStr}, le)
}

func TestParseFullInBoxError(t *testing.T) {
	errStr := "452 full Inbox"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrFullInbox, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseDailSMTPServerError(t *testing.T) {
	errStr := "Unexpected response dialing SMTP server"
	err := errors.New(errStr)
	le := ParseSMTPError(err)
	assert.Equal(t, &LookupError{Details: errStr, Message: errStr}, le)
}

func TestParseError_Code550(t *testing.T) {
	errStr := "550"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrServerUnavailable, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code400_Nil(t *testing.T) {
	errStr := "400"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, (*LookupError)(nil), le)
}

func TestParseError_Code401(t *testing.T) {
	errStr := "401"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, &LookupError{Details: errStr, Message: errStr}, le)
}

func TestParseError_Code421(t *testing.T) {
	errStr := "421"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrTryAgainLater, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code450(t *testing.T) {
	errStr := "450"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrMailboxBusy, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code451(t *testing.T) {
	errStr := "451"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrExceededMessagingLimits, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code452(t *testing.T) {
	errStr := "452"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrTooManyRCPT, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code503(t *testing.T) {
	errStr := "503"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrNeedMAILBeforeRCPT, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code551(t *testing.T) {
	errStr := "551"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrRCPTHasMoved, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code552(t *testing.T) {
	errStr := "552"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrFullInbox, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code553(t *testing.T) {
	errStr := "553"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrNoRelay, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_Code554(t *testing.T) {
	errStr := "554"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrNotAllowed, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_basicErr_timeout(t *testing.T) {
	errStr := "559 timeout"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrTimeout, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

func TestParseError_basicErr_blocked(t *testing.T) {
	errStr := "559 blocked"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrBlocked, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

// 450 4.1.1 is a transient (4xx) RCPT failure; even though the text says the
// mailbox "may not exist", 4xx responses are treated as temporary and mapped
// to ErrMailboxBusy rather than ErrMailboxNotFound.
func TestParseError_Code450_UnverifiedAddress(t *testing.T) {
	errStr := "450 4.1.1 <bert@example.com>: Recipient address rejected: unverified address: Mailbox might be disabled, full, or may not exist on the server. Reason: JFE030050"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrMailboxBusy, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

// 450 4.7.1 with "Greylisted" in the message should be treated as a
// greylisting/deferral situation and mapped to ErrTryAgainLater.
func TestParseError_Code450_Greylisted(t *testing.T) {
	errStr := "450 4.7.1 ralph@example.com: Recipient address rejected: Greylisted for 5 minutes"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrTryAgainLater, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

// A 250 2.1.0 reply is a successful SMTP response, so ParseSMTPError should
// not create a LookupError and instead return nil.
func TestParseError_250OK_Nil(t *testing.T) {
	errStr := "250 2.1.0 Ok"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, (*LookupError)(nil), le)
}

// 450 4.1.1 with "user unknown" in the text is still a 4xx transient error,
// so it is classified as ErrMailboxBusy, not ErrMailboxNotFound.
func TestParseError_450_UserUnknown(t *testing.T) {
	errStr := "450 4.1.1 <user@example.com>: user unknown"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrMailboxBusy, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

// 550 5.1.1 is a permanent (5xx) failure and the phrase "user unknown" is in
// the hard-bounce phrase list, so it is classified as ErrMailboxNotFound.
func TestParseError_550_UserUnknown(t *testing.T) {
	errStr := "550 5.1.1 <user@example.com>: user unknown"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrMailboxNotFound, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

// When the error string cannot be parsed as a status code, ParseSMTPError
// falls back to parseBasicErr, which maps "no such host" to ErrNoSuchHost.
func TestParseError_basicErr_noSuchHost(t *testing.T) {
	errStr := "dial tcp: lookup mx.example.com: no such host"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrNoSuchHost, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

// When there is no numeric status code, parseBasicErr inspects the text and
// maps messages containing "unavailable" to ErrServerUnavailable.
func TestParseError_basicErr_unavailable(t *testing.T) {
	errStr := "mail server unavailable"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrServerUnavailable, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

// A non-standard 499 status is treated as a 4xx temporary failure; since the
// message contains "timeout", the fallback parseBasicErr maps it to ErrTimeout.
func TestParseError_499Timeout(t *testing.T) {
	errStr := "499 timeout while reading from server"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, ErrTimeout, le.Message)
	assert.Equal(t, err.Error(), le.Details)
}

// 555 is an unmapped 5xx code without any special keywords, so after status
// parsing it falls back to parseBasicErr's default case, returning a
// LookupError with the raw string as both Message and Details.
func TestParseError_555Default(t *testing.T) {
	errStr := "555 unexpected server response"
	err := errors.New(errStr)
	le := ParseSMTPError(err)

	assert.Equal(t, &LookupError{Details: errStr, Message: errStr}, le)
}
