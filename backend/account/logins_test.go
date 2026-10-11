package account

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogins_ExpiredLoginCannotBeApproved(t *testing.T) {
	logins := &Logins{TTL: -time.Second, Limit: 10}
	code, err := logins.Start(Login{DeviceID: "1", DeviceUUID: "a"})
	require.NoError(t, err)

	assert.False(t, logins.Approve(code, User{Name: "boris"}))
}

func TestLogins_LimitIsEnforcedAndFreedByExpiry(t *testing.T) {
	logins := &Logins{TTL: time.Minute, Limit: 1}
	_, err := logins.Start(Login{DeviceID: "1", DeviceUUID: "a"})
	require.NoError(t, err)

	_, err = logins.Start(Login{DeviceID: "2", DeviceUUID: "b"})
	assert.ErrorIs(t, err, ErrTooManyLogins)

	logins.TTL = -time.Second
	for code, login := range logins.pending {
		login.Expires = time.Now().Add(-time.Second)
		logins.pending[code] = login
	}
	_, err = logins.Start(Login{DeviceID: "3", DeviceUUID: "c"})
	assert.NoError(t, err)
}

func TestLogins_StartIgnoresACallerSuppliedApproval(t *testing.T) {
	logins := &Logins{TTL: time.Minute, Limit: 10}
	code, err := logins.Start(Login{DeviceID: "1", DeviceUUID: "a", Approved: &User{Name: "root"}})
	require.NoError(t, err)

	_, claimed := logins.Claim(code, "1", "a")

	assert.False(t, claimed)
}
