package helper

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrInvalidCursor - cursor tidak bisa dipakai.
var ErrInvalidCursor = errors.New("cursor tidak valid")

// EncodeCursor membuat cursor base64 dari "UnixNano|ID".
func EncodeCursor(createdAt time.Time, id int) string {
	raw := fmt.Sprintf("%d|%d", createdAt.UnixNano(), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor mengembalikan (CreatedAt, ID) dari cursor.
func DecodeCursor(cursor string) (time.Time, int, error) {
	if cursor == "" {
		return time.Time{}, 0, ErrInvalidCursor
	}

	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, 0, ErrInvalidCursor
	}

	parts := strings.Split(string(raw), "|")
	if len(parts) != 2 {
		return time.Time{}, 0, ErrInvalidCursor
	}

	nano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, 0, ErrInvalidCursor
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, 0, ErrInvalidCursor
	}

	return time.Unix(0, nano).UTC(), id, nil
}
