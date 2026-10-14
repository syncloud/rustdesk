package account

import "time"

type Devices struct {
	Store     Store
	Limit     int
	SeenEvery time.Duration
}

func (d *Devices) Report(report Device) (bool, error) {
	report.LastSeen = time.Now()
	return d.Store.SaveDevice(report, d.Limit)
}

func (d *Devices) Seen(id, uuid string) (bool, error) {
	device, found, err := d.Store.Device(id)
	if err != nil || !found || device.UUID != uuid {
		return false, err
	}
	now := time.Now()
	if now.Sub(device.LastSeen) < d.SeenEvery {
		return true, nil
	}
	return true, d.Store.SetLastSeen(id, now)
}

func (d *Devices) Assign(id, owner string) error {
	return d.Store.SetOwner(id, owner)
}

func (d *Devices) List() ([]Device, error) {
	return d.Store.Devices()
}

func (d *Devices) OwnedBy(owner string) ([]Device, error) {
	return d.Store.DevicesOwnedBy(owner)
}
