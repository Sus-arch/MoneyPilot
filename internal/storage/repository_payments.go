package storage

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

// SavePaymentByClientIdAndBank сохраняет платеж в БД
func (r *Repository) SavePaymentByClientIdAndBank(
	clientID, bankCode, paymentID, debtorAccount, creditorAccount string,
	creditorBankCode *string,
	amount float64, currency string,
	comment, description *string,
	paymentConsentID *string,
	status string,
) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var bankID int
	if err := tx.QueryRow(`SELECT id FROM banks WHERE code=$1`, bankCode).Scan(&bankID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("bank not found")
		}
		return err
	}

	var userID int
	if err := tx.QueryRow(`SELECT id FROM users WHERE client_id=$1 AND bank_id=$2`, clientID, bankID).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found for this bank")
		}
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO payments (
			payment_id, user_id, bank_id, debtor_account, creditor_account,
			creditor_bank_code, amount, currency, comment, description,
			payment_consent_id, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		paymentID, userID, bankID, debtorAccount, creditorAccount,
		creditorBankCode, amount, currency, comment, description,
		paymentConsentID, status)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetPaymentByPaymentID получает платеж по payment_id
func (r *Repository) GetPaymentByPaymentID(paymentID string) (*Payment, error) {
	var p Payment
	var creditorBankCode, comment, description, paymentConsentID sql.NullString

	err := r.DB.QueryRow(`
		SELECT id, payment_id, user_id, bank_id, debtor_account, creditor_account,
			creditor_bank_code, amount, currency, comment, description,
			payment_consent_id, status, created_at, updated_at
		FROM payments
		WHERE payment_id = $1
	`, paymentID).Scan(
		&p.ID, &p.PaymentID, &p.UserID, &p.BankID, &p.DebtorAccount, &p.CreditorAccount,
		&creditorBankCode, &p.Amount, &p.Currency, &comment, &description,
		&paymentConsentID, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if creditorBankCode.Valid {
		p.CreditorBankCode = &creditorBankCode.String
	}
	if comment.Valid {
		p.Comment = &comment.String
	}
	if description.Valid {
		p.Description = &description.String
	}
	if paymentConsentID.Valid {
		p.PaymentConsentID = &paymentConsentID.String
	}

	return &p, nil
}

// GetPaymentConsentByConsentID получает согласие на платеж по consent_id или request_id
func (r *Repository) GetPaymentConsentByConsentID(consentID string) (*PaymentConsent, error) {
	var c PaymentConsent
	var consentIDPtr, requestingBank, currency, creditorAccount, creditorName, reference, reason sql.NullString
	var amount, maxAmountPerPayment, maxTotalAmount, vrpMaxIndividualAmount, vrpDailyLimit, vrpMonthlyLimit sql.NullFloat64
	var maxUses sql.NullInt64
	var validFrom, validUntil, expiresAt sql.NullTime

	err := r.DB.QueryRow(`
		SELECT 
			pc.id, pc.request_id, pc.consent_id, pc.user_id, pc.bank_id, 
			pc.requesting_bank, pc.consent_type,
			pc.amount, pc.currency, pc.debtor_account, pc.creditor_account,
			pc.creditor_name, pc.reference, pc.max_uses, pc.max_amount_per_payment,
			pc.max_total_amount, pc.allowed_creditor_accounts,
			pc.vrp_max_individual_amount, pc.vrp_daily_limit, pc.vrp_monthly_limit,
			pc.valid_from, pc.valid_until, pc.reason, pc.status, pc.expires_at,
			pc.created_at, pc.updated_at
		FROM payment_consents pc
		WHERE (pc.consent_id = $1 OR pc.request_id = $1) AND pc.status = 'approved'
	`, consentID).Scan(
		&c.ID, &c.RequestID, &consentIDPtr, &c.UserID, &c.BankID, &requestingBank,
		&c.ConsentType, &amount, &currency, &c.DebtorAccount, &creditorAccount,
		&creditorName, &reference, &maxUses, &maxAmountPerPayment, &maxTotalAmount,
		pq.Array(&c.AllowedCreditorAccounts),
		&vrpMaxIndividualAmount, &vrpDailyLimit, &vrpMonthlyLimit,
		&validFrom, &validUntil, &reason, &c.Status, &expiresAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if consentIDPtr.Valid {
		c.ConsentID = &consentIDPtr.String
	}
	if requestingBank.Valid {
		c.RequestingBank = &requestingBank.String
	}
	if currency.Valid {
		c.Currency = &currency.String
	}
	if creditorAccount.Valid {
		c.CreditorAccount = &creditorAccount.String
	}
	if creditorName.Valid {
		c.CreditorName = &creditorName.String
	}
	if reference.Valid {
		c.Reference = &reference.String
	}
	if reason.Valid {
		c.Reason = &reason.String
	}
	if amount.Valid {
		val := amount.Float64
		c.Amount = &val
	}
	if maxUses.Valid {
		val := int(maxUses.Int64)
		c.MaxUses = &val
	}
	if maxAmountPerPayment.Valid {
		val := maxAmountPerPayment.Float64
		c.MaxAmountPerPayment = &val
	}
	if maxTotalAmount.Valid {
		val := maxTotalAmount.Float64
		c.MaxTotalAmount = &val
	}
	if vrpMaxIndividualAmount.Valid {
		val := vrpMaxIndividualAmount.Float64
		c.VRPMaxIndividualAmount = &val
	}
	if vrpDailyLimit.Valid {
		val := vrpDailyLimit.Float64
		c.VRPDailyLimit = &val
	}
	if vrpMonthlyLimit.Valid {
		val := vrpMonthlyLimit.Float64
		c.VRPMonthlyLimit = &val
	}
	if validFrom.Valid {
		c.ValidFrom = &validFrom.Time
	}
	if validUntil.Valid {
		c.ValidUntil = &validUntil.Time
	}
	if expiresAt.Valid {
		c.ExpiresAt = &expiresAt.Time
	}

	return &c, nil
}
