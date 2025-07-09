package uuid

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

// UUID is a thin wrapper around google's uuid.UUID that implements database
// scanning and driver.Value interfaces.
type UUID uuid.UUID

// NewUUID creates a new random UUID.
func NewUUID() UUID {
	return UUID(uuid.New())
}

func (u UUID) String() string {
	return uuid.UUID(u).String()
}

func (u *UUID) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("UUID.Scan: expected []byte, got %T", src)
	}
	id, err := uuid.FromBytes(b)
	if err != nil {
		return err
	}
	*u = UUID(id)
	return nil
}

func (u UUID) Value() (driver.Value, error) {
	return uuid.UUID(u).MarshalBinary()
}

func (u UUID) MarshalText() ([]byte, error) {
	return []byte(uuid.UUID(u).String()), nil
}

func (u *UUID) UnmarshalText(text []byte) error {
	parsed, err := uuid.ParseBytes(text)
	if err != nil {
		return err
	}
	*u = UUID(parsed)
	return nil
}
