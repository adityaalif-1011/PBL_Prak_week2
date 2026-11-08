package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager menjalankan fungsi di dalam satu transaksi PostgreSQL.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithTx menjalankan fn di dalam satu transaksi.
// Rollback dipanggil via defer sehingga commit yang gagal atau panic
// tetap mengembalikan koneksi ke pool tanpa meninggalkan transaksi
// menggantung. Rollback menjadi no-op bila fn sudah sukses dan Commit
// dipanggil.
func (m *TxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // nolint:errcheck // no-op setelah Commit

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// WithTxOnce menjalankan fn di dalam satu transaksi TANPA retry loop.
// Berbeda dengan WithTx yang bisa dipakai pembungkus retry, versi ini
// selalu melakukan tepat satu percobaan: Begin → fn → Commit (atau
// Rollback otomatis via defer bila fn error).
// Dipakai oleh worker/relay yang ingin kontrol penuh atas retry mereka
// sendiri (mis. lewat mekanisme retry Asynq).
func (m *TxManager) WithTxOnce(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // nolint:errcheck // no-op setelah Commit

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
