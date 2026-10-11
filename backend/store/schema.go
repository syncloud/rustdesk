package store

var schema = []string{
	`CREATE TABLE tokens (
		hash TEXT PRIMARY KEY,
		user_name TEXT NOT NULL,
		user_email TEXT NOT NULL,
		device_id TEXT NOT NULL,
		device_uuid TEXT NOT NULL,
		created INTEGER NOT NULL
	);
	CREATE INDEX tokens_device ON tokens (device_id, device_uuid);
	CREATE TABLE devices (
		id TEXT PRIMARY KEY,
		uuid TEXT NOT NULL,
		hostname TEXT NOT NULL,
		os TEXT NOT NULL,
		username TEXT NOT NULL,
		version TEXT NOT NULL,
		owner TEXT NOT NULL,
		last_seen INTEGER NOT NULL
	);
	CREATE INDEX devices_owner ON devices (owner, last_seen);
	CREATE TABLE address_books (
		user TEXT PRIMARY KEY,
		data TEXT NOT NULL
	);`,
}
