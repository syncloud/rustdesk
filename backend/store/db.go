package store

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"backend/account"
)

const (
	file          = "account.db"
	deviceColumns = "id, uuid, hostname, os, username, version, owner, last_seen"
)

type DB struct {
	db *sql.DB
}

func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, file)
	created, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := created.Close(); err != nil {
		return nil, err
	}
	pragmas := url.Values{"_pragma": {"busy_timeout(5000)", "journal_mode(WAL)", "synchronous(NORMAL)"}}
	db, err := sql.Open("sqlite", "file:"+path+"?"+pragmas.Encode())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) SaveToken(hash string, token account.Token) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"DELETE FROM tokens WHERE device_id = ? AND device_uuid = ?", token.DeviceID, token.DeviceUUID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"INSERT INTO tokens (hash, user_name, user_email, device_id, device_uuid, created) VALUES (?, ?, ?, ?, ?, ?)",
		hash, token.User.Name, token.User.Email, token.DeviceID, token.DeviceUUID, token.Created.UnixNano(),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) Token(hash string) (account.Token, bool, error) {
	token := account.Token{}
	created := int64(0)
	err := d.db.QueryRow(
		"SELECT user_name, user_email, device_id, device_uuid, created FROM tokens WHERE hash = ?", hash,
	).Scan(&token.User.Name, &token.User.Email, &token.DeviceID, &token.DeviceUUID, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return account.Token{}, false, nil
	}
	if err != nil {
		return account.Token{}, false, err
	}
	token.Created = time.Unix(0, created).UTC()
	return token, true, nil
}

func (d *DB) DeleteToken(hash string) error {
	_, err := d.db.Exec("DELETE FROM tokens WHERE hash = ?", hash)
	return err
}

func (d *DB) SaveDevice(device account.Device, limit int) (bool, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	uuid := ""
	err = tx.QueryRow("SELECT uuid FROM devices WHERE id = ?", device.ID).Scan(&uuid)
	known := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if known && uuid != device.UUID {
		return false, nil
	}
	if !known {
		if _, err := tx.Exec(
			`DELETE FROM devices WHERE (SELECT COUNT(*) FROM devices) >= ? AND id =
				(SELECT id FROM devices WHERE owner = '' ORDER BY last_seen LIMIT 1)`,
			limit,
		); err != nil {
			return false, err
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO devices (`+deviceColumns+`) VALUES (?, ?, ?, ?, ?, ?, '', ?)
			ON CONFLICT (id) DO UPDATE SET hostname = excluded.hostname, os = excluded.os,
				username = excluded.username, version = excluded.version, last_seen = excluded.last_seen`,
		device.ID, device.UUID, device.Hostname, device.OS, device.Username, device.Version,
		device.LastSeen.UnixNano(),
	); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (d *DB) Device(id string) (account.Device, bool, error) {
	devices, err := d.devices("SELECT "+deviceColumns+" FROM devices WHERE id = ?", id)
	if err != nil || len(devices) == 0 {
		return account.Device{}, false, err
	}
	return devices[0], true, nil
}

func (d *DB) SetLastSeen(id string, at time.Time) error {
	_, err := d.db.Exec("UPDATE devices SET last_seen = ? WHERE id = ?", at.UnixNano(), id)
	return err
}

func (d *DB) SetOwner(id, owner string) error {
	_, err := d.db.Exec("UPDATE devices SET owner = ? WHERE id = ?", owner, id)
	return err
}

func (d *DB) Devices() ([]account.Device, error) {
	return d.devices("SELECT " + deviceColumns + " FROM devices ORDER BY id")
}

func (d *DB) DevicesOwnedBy(owner string) ([]account.Device, error) {
	return d.devices("SELECT "+deviceColumns+" FROM devices WHERE owner = ? ORDER BY id", owner)
}

func (d *DB) AddressBook(user string) (string, error) {
	data := ""
	err := d.db.QueryRow("SELECT data FROM address_books WHERE user = ?", user).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return data, err
}

func (d *DB) SaveAddressBook(user, data string) error {
	_, err := d.db.Exec(
		"INSERT INTO address_books (user, data) VALUES (?, ?) ON CONFLICT (user) DO UPDATE SET data = excluded.data",
		user, data,
	)
	return err
}

func (d *DB) devices(query string, args ...any) ([]account.Device, error) {
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	devices := []account.Device{}
	for rows.Next() {
		device := account.Device{}
		lastSeen := int64(0)
		if err := rows.Scan(
			&device.ID, &device.UUID, &device.Hostname, &device.OS, &device.Username,
			&device.Version, &device.Owner, &lastSeen,
		); err != nil {
			return nil, err
		}
		device.LastSeen = time.Unix(0, lastSeen).UTC()
		devices = append(devices, device)
	}
	return devices, rows.Err()
}

func migrate(db *sql.DB) error {
	version := 0
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > len(schema) {
		return fmt.Errorf("database schema %d is newer than this version supports (%d)", version, len(schema))
	}
	for ; version < len(schema); version++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(schema[version]); err != nil {
			tx.Rollback()
			return fmt.Errorf("schema step %d: %w", version+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
