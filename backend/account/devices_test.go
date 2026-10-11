package account_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend/account"
)

func TestDevices_OldestUnownedDeviceIsForgottenAtTheLimit(t *testing.T) {
	devices := &account.Devices{Store: open(t, t.TempDir()), Limit: 2, SeenEvery: time.Hour}
	for _, id := range []string{"1", "2"} {
		_, err := devices.Report(account.Device{ID: id, UUID: id})
		require.NoError(t, err)
	}
	require.NoError(t, devices.Assign("1", "boris"))

	_, err := devices.Report(account.Device{ID: "3", UUID: "3"})
	require.NoError(t, err)

	listed, err := devices.List()
	require.NoError(t, err)
	ids := []string{}
	for _, device := range listed {
		ids = append(ids, device.ID)
	}
	assert.Equal(t, []string{"1", "3"}, ids)
}

func TestDevices_OwnerSurvivesANewReportAndARestart(t *testing.T) {
	dir := t.TempDir()
	devices := &account.Devices{Store: open(t, dir), Limit: 10, SeenEvery: time.Hour}
	_, err := devices.Report(account.Device{ID: "1", UUID: "a", Hostname: "old"})
	require.NoError(t, err)
	require.NoError(t, devices.Assign("1", "boris"))
	_, err = devices.Report(account.Device{ID: "1", UUID: "a", Hostname: "new"})
	require.NoError(t, err)

	reloaded := &account.Devices{Store: open(t, dir), Limit: 10, SeenEvery: time.Hour}

	owned, err := reloaded.OwnedBy("boris")
	require.NoError(t, err)
	require.Len(t, owned, 1)
	assert.Equal(t, "new", owned[0].Hostname)
}

func TestDevices_AnotherUUIDCannotTakeOverAnID(t *testing.T) {
	devices := &account.Devices{Store: open(t, t.TempDir()), Limit: 10, SeenEvery: time.Hour}
	_, err := devices.Report(account.Device{ID: "1", UUID: "a", Hostname: "mine"})
	require.NoError(t, err)

	accepted, err := devices.Report(account.Device{ID: "1", UUID: "b", Hostname: "theirs"})
	require.NoError(t, err)
	seen, err := devices.Seen("1", "b")
	require.NoError(t, err)

	listed, err := devices.List()
	require.NoError(t, err)
	assert.False(t, accepted)
	assert.False(t, seen)
	assert.Equal(t, "mine", listed[0].Hostname)
}

func TestDevices_HeartbeatMovesLastSeenOnceTheIntervalHasPassed(t *testing.T) {
	devices := &account.Devices{Store: open(t, t.TempDir()), Limit: 10, SeenEvery: 0}
	_, err := devices.Report(account.Device{ID: "1", UUID: "a"})
	require.NoError(t, err)
	before, err := devices.List()
	require.NoError(t, err)

	seen, err := devices.Seen("1", "a")
	require.NoError(t, err)

	after, err := devices.List()
	require.NoError(t, err)
	assert.True(t, seen)
	assert.True(t, after[0].LastSeen.After(before[0].LastSeen))
}
