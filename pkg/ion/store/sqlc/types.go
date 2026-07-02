package sqlc

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringList stores a JSON-encoded string slice in SQLite.
type StringList []string

// Scan decodes a SQLite TEXT or BLOB value into a string list.
func (l *StringList) Scan(value any) error {
	if value == nil {
		*l = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("scan string list from %T: unsupported value", value)
	}

	if err := json.Unmarshal(data, l); err != nil {
		return fmt.Errorf("decode string list: %w", err)
	}

	return nil
}

// Value encodes a string list as JSON for SQLite.
func (l StringList) Value() (driver.Value, error) {
	data, err := json.Marshal([]string(l))
	if err != nil {
		return nil, fmt.Errorf("encode string list: %w", err)
	}

	return string(data), nil
}
