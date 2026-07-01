package persistence

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	domb "github.com/retechfin/retechfin-api/internal/domain/budget"
	domh "github.com/retechfin/retechfin-api/internal/domain/health"
	doml "github.com/retechfin/retechfin-api/internal/domain/ledger"
	"gorm.io/gorm"
)

func mapLedgerErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return doml.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return doml.ErrConflict
	}
	return err
}

func mapHealthErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domh.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return domh.ErrConflict
	}
	return err
}

func mapBudgetErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domb.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return domb.ErrConflict
	}
	return err
}
