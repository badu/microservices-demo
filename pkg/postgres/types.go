package postgres

import (
	"database/sql"
	"encoding/json"
)

type NullString sql.NullString

func (n *NullString) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.String)
	} else {
		return json.Marshal(nil)
	}
}

func (n *NullString) UnmarshalJSON(bytes []byte) error {
	var s *string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	if s != nil {
		n.Valid = true
		n.String = *s
	} else {
		n.Valid = false
	}
	return nil
}

type NullFloat64 sql.NullFloat64

func (n *NullFloat64) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Float64)
	} else {
		return json.Marshal(nil)
	}
}

func (n *NullFloat64) UnmarshalJSON(bytes []byte) error {
	var v *float64
	if err := json.Unmarshal(bytes, &v); err != nil {
		return err
	}

	if v != nil {
		n.Valid = true
		n.Float64 = *v
	} else {
		n.Valid = false
	}
	return nil
}
