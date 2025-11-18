package emailverifier

import (
	"errors"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckSMTPUnSupportedVendor(t *testing.T) {
	err := verifier.EnableAPIVerifier("unsupported_vendor")
	assert.Error(t, err)
}

func TestCheckSMTPOK_ByApi(t *testing.T) {
	cases := []struct {
		name     string
		domain   string
		username string
		expected *SMTP
	}{
		{
			name:     "yahoo exists",
			domain:   "yahoo.com",
			username: "someone",
			expected: &SMTP{
				HostExists:  true,
				Deliverable: true,
			},
		},
		{
			name:     "myyahoo exists",
			domain:   "myyahoo.com",
			username: "someone",
			expected: &SMTP{
				HostExists:  true,
				Deliverable: true,
			},
		},
		{
			name:     "yahoo no exists",
			domain:   "yahoo.com",
			username: "123",
			expected: &SMTP{
				HostExists:  true,
				Deliverable: false,
			},
		},
		{
			name:     "myyahoo no exists",
			domain:   "myyahoo.com",
			username: "123",
			expected: &SMTP{
				HostExists:  true,
				Deliverable: false,
			},
		},
	}
	_ = verifier.EnableAPIVerifier(YAHOO)
	defer verifier.DisableAPIVerifier(YAHOO)
	for _, c := range cases {
		test := c
		t.Run(test.name, func(tt *testing.T) {
			smtp, err := verifier.CheckSMTP(test.domain, test.username)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, smtp)
		})
	}
}

func TestCheckSMTPOK_HostExists(t *testing.T) {
	domain := "github.com"

	smtp, err := verifier.CheckSMTP(domain, "")
	expected := SMTP{
		HostExists: true,
		FullInbox:  false,
		CatchAll:   true,
		Disabled:   false,
	}
	assert.NoError(t, err)
	assert.Equal(t, &expected, smtp)
}

func TestCheckSMTPOK_CatchAllHost(t *testing.T) {
	domain := "gmail.com"

	smtp, err := verifier.CheckSMTP(domain, "")
	expected := SMTP{
		HostExists: true,
		FullInbox:  false,
		CatchAll:   false,
		Disabled:   false,
	}
	assert.NoError(t, err)
	assert.Equal(t, &expected, smtp)
}

func TestCheckSMTPOK_NoCatchAllHost(t *testing.T) {
	domain := "gmail.com"

	smtp, err := verifier.CheckSMTP(domain, "")
	expected := SMTP{
		HostExists: true,
		FullInbox:  false,
		CatchAll:   false,
		Disabled:   false,
	}
	assert.NoError(t, err)
	assert.Equal(t, &expected, smtp)
}

func TestCheckSMTPOK_NoCatchAllHostCatchAllCheckDisabled(t *testing.T) {
	domain := "gmail.com"

	var verifier = NewVerifier().EnableSMTPCheck().DisableCatchAllCheck()
	smtp, err := verifier.CheckSMTP(domain, "")
	expected := SMTP{
		HostExists: true,
		FullInbox:  false,
		CatchAll:   true,
		Disabled:   false,
	}
	assert.NoError(t, err)
	assert.Equal(t, &expected, smtp)
}

func TestCheckSMTPOK_UpdateFromEmail(t *testing.T) {
	domain := "github.com"
	verifier.FromEmail("from@email.top")

	smtp, err := verifier.CheckSMTP(domain, "")
	expected := SMTP{
		HostExists:  true,
		FullInbox:   false,
		CatchAll:    true,
		Deliverable: false,
		Disabled:    false,
	}
	assert.NoError(t, err)
	assert.Equal(t, &expected, smtp)
}

func TestCheckSMTPOK_UpdateHelloName(t *testing.T) {
	domain := "github.com"
	verifier.HelloName("email.top")

	smtp, err := verifier.CheckSMTP(domain, "")
	expected := SMTP{
		HostExists:  true,
		FullInbox:   false,
		CatchAll:    true,
		Deliverable: false,
		Disabled:    false,
	}
	assert.NoError(t, err)
	assert.Equal(t, &expected, smtp)
}

func TestCheckSMTPOK_WithNoExistUsername(t *testing.T) {
	domain := "github.com"
	username := "testing"

	smtp, err := verifier.CheckSMTP(domain, username)
	expected := SMTP{
		HostExists: true,
		FullInbox:  false,
		CatchAll:   true,
		Disabled:   false,
	}
	assert.NoError(t, err)
	assert.Equal(t, &expected, smtp)
}

func TestCheckSMTP_DisabledSMTPCheck(t *testing.T) {
	domain := "github.com"

	verifier.DisableSMTPCheck()
	smtp, err := verifier.CheckSMTP(domain, "username")
	verifier.EnableSMTPCheck()

	assert.NoError(t, err)
	assert.Nil(t, smtp)
}

func TestCheckSMTPOK_HostNotExists(t *testing.T) {
	domain := "notExistHost.com"

	smtp, err := verifier.CheckSMTP(domain, "")
	assert.Error(t, err, ErrNoSuchHost)
	assert.Equal(t, &SMTP{}, smtp)
}

func TestNewSMTPClientOK(t *testing.T) {
	domain := "gmail.com"
	timeout := 5 * time.Second
	ret, _, err := newSMTPClientWithStrategy(domain, "", timeout, timeout, MXStrategyFirstConnected)
	assert.NotNil(t, ret)
	assert.Nil(t, err)
}

func TestNewSMTPClientFailed_WithInvalidProxy(t *testing.T) {
	domain := "gmail.com"
	proxyURI := "socks5://user:password@127.0.0.1:1080?timeout=5s"
	timeout := 5 * time.Second
	ret, _, err := newSMTPClientWithStrategy(domain, proxyURI, timeout, timeout, MXStrategyFirstConnected)
	assert.Nil(t, ret)
	assert.Error(t, err, syscall.ECONNREFUSED)
}

func TestNewSMTPClientFailed(t *testing.T) {
	domain := "zzzz171777.com"
	timeout := 5 * time.Second
	ret, _, err := newSMTPClientWithStrategy(domain, "", timeout, timeout, MXStrategyFirstConnected)
	assert.Nil(t, ret)
	assert.Error(t, err)
}

func TestDialSMTPFailed_NoPortIsConfigured(t *testing.T) {
	disposableDomain := "zzzz1717.com"
	timeout := 5 * time.Second
	ret, err := dialSMTP(disposableDomain, "", timeout, timeout)
	assert.Nil(t, ret)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "missing port"))
}

func TestDialSMTPFailed_NoSuchHost(t *testing.T) {
	disposableDomain := "zzzzyyyyaaa123.com:25"
	timeout := 5 * time.Second
	ret, err := dialSMTP(disposableDomain, "", timeout, timeout)
	assert.Nil(t, ret)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such host"))
}

func TestNewSMTPClientPriority_UsesLowestPreferenceWhenAvailable(t *testing.T) {
	originalDialSMTP := dialSMTPFunc
	defer func() {
		dialSMTPFunc = originalDialSMTP
	}()

	mxRecords := []*net.MX{
		{Host: "primary.example.com.", Pref: 0},
		{Host: "backup.example.com.", Pref: 10},
	}

	dialSMTPFunc = func(addr, proxyURI string, connectTimeout, operationTimeout time.Duration) (*smtp.Client, error) {
		if strings.Contains(addr, "backup.example.com.") {
			t.Fatalf("should not dial lower-priority MX: %s", addr)
		}
		return &smtp.Client{}, nil
	}

	client, mx, err := newSMTPClientPriority(mxRecords, "", 1*time.Second, 1*time.Second)
	assert.NoError(t, err)
	if assert.NotNil(t, client) && assert.NotNil(t, mx) {
		assert.Equal(t, "primary.example.com.", mx.Host)
	}
}

func TestNewSMTPClientPriority_FallsBackToHigherPreferenceOnFailure(t *testing.T) {
	originalDialSMTP := dialSMTPFunc
	defer func() {
		dialSMTPFunc = originalDialSMTP
	}()

	mxRecords := []*net.MX{
		{Host: "primary.example.com.", Pref: 0},
		{Host: "backup.example.com.", Pref: 10},
	}

	dialSMTPFunc = func(addr, proxyURI string, connectTimeout, operationTimeout time.Duration) (*smtp.Client, error) {
		switch {
		case strings.Contains(addr, "primary.example.com."):
			return nil, errors.New("primary MX failure")
		case strings.Contains(addr, "backup.example.com."):
			return &smtp.Client{}, nil
		default:
			return nil, errors.New("unexpected host")
		}
	}

	client, mx, err := newSMTPClientPriority(mxRecords, "", 1*time.Second, 1*time.Second)
	assert.NoError(t, err)
	if assert.NotNil(t, client) && assert.NotNil(t, mx) {
		assert.Equal(t, "backup.example.com.", mx.Host)
	}
}

func TestNewSMTPClientPriority_SamePreferenceGroupBeforeHigherPreference(t *testing.T) {
	originalDialSMTP := dialSMTPFunc
	defer func() {
		dialSMTPFunc = originalDialSMTP
	}()

	mxRecords := []*net.MX{
		{Host: "host1.example.com.", Pref: 0},
		{Host: "host2.example.com.", Pref: 0},
		{Host: "host3.example.com.", Pref: 10},
	}

	var mu sync.Mutex
	host1Called := false
	host2Called := false

	dialSMTPFunc = func(addr, proxyURI string, connectTimeout, operationTimeout time.Duration) (*smtp.Client, error) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case strings.Contains(addr, "host1.example.com."):
			host1Called = true
			return nil, errors.New("host1 failure")
		case strings.Contains(addr, "host2.example.com."):
			host2Called = true
			return nil, errors.New("host2 failure")
		case strings.Contains(addr, "host3.example.com."):
			if !(host1Called && host2Called) {
				t.Fatalf("higher-preference MX dialed before all equal-preference hosts")
			}
			return &smtp.Client{}, nil
		default:
			return nil, errors.New("unexpected host")
		}
	}

	client, mx, err := newSMTPClientPriority(mxRecords, "", 1*time.Second, 1*time.Second)
	assert.NoError(t, err)
	if assert.NotNil(t, client) && assert.NotNil(t, mx) {
		assert.Equal(t, "host3.example.com.", mx.Host)
	}

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, host1Called, "host1 (pref 0) should be dialed before fallback")
	assert.True(t, host2Called, "host2 (pref 0) should be dialed before fallback")
}

func TestNewSMTPClientWithStrategy_Priority_RespectsMXPreference(t *testing.T) {
	originalLookupMX := lookupMX
	originalDialSMTP := dialSMTPFunc
	defer func() {
		lookupMX = originalLookupMX
		dialSMTPFunc = originalDialSMTP
	}()

	lookupMX = func(domain string) ([]*net.MX, error) {
		if domain != "example.com" {
			t.Fatalf("unexpected domain: %s", domain)
		}
		return []*net.MX{
			{Host: "primary.example.com.", Pref: 0},
			{Host: "backup.example.com.", Pref: 10},
		}, nil
	}

	dialSMTPFunc = func(addr, proxyURI string, connectTimeout, operationTimeout time.Duration) (*smtp.Client, error) {
		if strings.Contains(addr, "backup.example.com.") {
			t.Fatalf("should not dial lower-priority MX when primary is available: %s", addr)
		}
		return &smtp.Client{}, nil
	}

	client, mx, err := newSMTPClientWithStrategy("example.com", "", 1*time.Second, 1*time.Second, MXStrategyPriority)
	assert.NoError(t, err)
	if assert.NotNil(t, client) && assert.NotNil(t, mx) {
		assert.Equal(t, "primary.example.com.", mx.Host)
	}
}
