package db

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"github.com/ZhdanovichVlad/go-katas/checking_payment_status_via_API/internal/domain"
)

type storage struct {
	db *sql.DB
}

func New(db *sql.DB) *storage {
	return &storage{db: db}
}

func (s *storage) GetPendingPaymentsAndLockRows(ctx context.Context) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
	SELECT vendor_tx_id
	FROM payments
	WHERE status = 'pending_vendor' AND (is_locked = FALSE OR locked_at < now() - interval '1 hour') 
	order by updated_at
	Limit 10
	FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	defer rows.Close()

	var batch []string

	for rows.Next() {

		var p string

		if err := rows.Scan(&p); err != nil {

			return nil, err

		}

		batch = append(batch, p)

	}

	if err := rows.Err(); err != nil {

		return nil, err

	}

	stmt, err = tx.PrepareContext(ctx, `
	UPDATE payments SET is_locked = TRUE, time_locked = NOW()
	WHERE vendor_tx_id = ANY($1)`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, pq.Array(batch))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return batch, nil
}

func (s *storage) UpdatePaymentStatusAndUnlockRows(ctx context.Context, vendorTxID []string, status domain.PaymentStatus) error {
	stmt, err := s.db.PrepareContext(ctx, `
	UPDATE payments set status = $1, updated_at = NOW(), is_locked = FALSE, time_locked = NULL
	 WHERE vendor_tx_id = ANY($2)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, status, pq.Array(vendorTxID))

	if err != nil {
		return err
	}
	return nil

}
