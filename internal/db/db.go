package db

import (
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func NullUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func ValidUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func NullText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func ValidText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func NullBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

func UUIDPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

func TextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func BoolVal(b pgtype.Bool) bool {
	return b.Valid && b.Bool
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
