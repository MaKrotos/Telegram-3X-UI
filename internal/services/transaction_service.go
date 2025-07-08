package services

import (
	"database/sql"
	"time"
)

type Transaction struct {
	ID                      int
	TelegramPaymentChargeID string
	TelegramUserID          int64
	Amount                  int
	InvoicePayload          string
	Status                  string
	Type                    string // 'payment' или 'refund'
	Reason                  string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type TransactionService struct {
	db *sql.DB
}

func NewTransactionService(db *sql.DB) *TransactionService {
	return &TransactionService{db: db}
}

func (s *TransactionService) AddTransaction(tx *Transaction) error {
	_, err := s.db.Exec(`
		INSERT INTO transactions (
			telegram_payment_charge_id, telegram_user_id, amount, invoice_payload, status, type, reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`,
		tx.TelegramPaymentChargeID,
		tx.TelegramUserID,
		tx.Amount,
		tx.InvoicePayload,
		tx.Status,
		tx.Type,
		tx.Reason,
	)
	return err
}

func (s *TransactionService) GetAllTransactions() ([]*Transaction, error) {
	rows, err := s.db.Query(`
		SELECT id, telegram_payment_charge_id, telegram_user_id, amount, invoice_payload, status, type, reason, created_at, updated_at
		FROM transactions
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*Transaction
	for rows.Next() {
		tx := &Transaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.TelegramPaymentChargeID,
			&tx.TelegramUserID,
			&tx.Amount,
			&tx.InvoicePayload,
			&tx.Status,
			&tx.Type,
			&tx.Reason,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}
	return transactions, nil
}
