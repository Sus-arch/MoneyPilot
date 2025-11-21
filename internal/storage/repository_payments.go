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

// GetAccountByIdentificationAndBank получает счет по identification и bank_id/user_id
func (r *Repository) GetAccountByIdentificationAndBank(identification string, userID, bankID int) (*Account, error) {
	var acc Account
	var accountSubType, nickname, ownerName, schemeName, identificationField sql.NullString
	var openingDate sql.NullTime

	err := r.DB.QueryRow(`
		SELECT id, user_id, bank_id, account_number, account_type, account_subtype,
			nickname, currency, balance, status, owner_name, opening_date,
			scheme_name, identification, created_at, updated_at
		FROM accounts 
		WHERE identification = $1 AND user_id = $2 AND bank_id = $3
	`, identification, userID, bankID).Scan(
		&acc.ID, &acc.UserID, &acc.BankID, &acc.AccountNumber, &acc.AccountType,
		&accountSubType, &nickname, &acc.Currency, &acc.Balance, &acc.Status,
		&ownerName, &openingDate, &schemeName, &identificationField,
		&acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if accountSubType.Valid {
		acc.AccountSubType = &accountSubType.String
	}
	if nickname.Valid {
		acc.Nickname = &nickname.String
	}
	if ownerName.Valid {
		acc.OwnerName = &ownerName.String
	}
	if openingDate.Valid {
		acc.OpeningDate = &openingDate.Time
	}
	if schemeName.Valid {
		acc.SchemeName = &schemeName.String
	}
	if identificationField.Valid {
		acc.Identification = &identificationField.String
	}

	return &acc, nil
}

// FindMatchingPaymentConsents находит подходящие согласия для платежа
// Проверяет как по account_id (debtor_account в БД), так и по identification
func (r *Repository) FindMatchingPaymentConsents(userID, bankID int, debtorAccountIdentification, creditorAccount string, amount float64, currency string) ([]PaymentConsent, error) {
	// Ищем согласия, которые подходят для данного платежа
	// debtor_account в payment_consents может быть либо account_id (например, "acc-1901"), 
	// либо identification (например, "4081781008101077872")
	// Нужно проверить оба варианта через JOIN с accounts
	query := `
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
		WHERE pc.user_id = $1 
		  AND pc.bank_id = $2
		  AND pc.status = 'approved'
		  AND (pc.valid_until IS NULL OR pc.valid_until > NOW())
		  AND (
			-- Проверяем прямое совпадение (если debtor_account это identification)
			pc.debtor_account = $3
			OR
			-- Проверяем через account_number (если debtor_account это account_id)
			-- Находим счет по identification и проверяем, что его account_number совпадает с debtor_account в согласии
			EXISTS (
				SELECT 1 FROM accounts a 
				WHERE a.user_id = $1 
				  AND a.bank_id = $2 
				  AND a.identification = $3
				  AND a.account_number = pc.debtor_account
			)
		  )
	`

	rows, err := r.DB.Query(query, userID, bankID, debtorAccountIdentification)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []PaymentConsent
	for rows.Next() {
		var c PaymentConsent
		var consentIDPtr, requestingBank, currencyField, creditorAccountField, creditorName, reference, reason sql.NullString
		var amountField, maxAmountPerPayment, maxTotalAmount, vrpMaxIndividualAmount, vrpDailyLimit, vrpMonthlyLimit sql.NullFloat64
		var maxUses sql.NullInt64
		var validFrom, validUntil, expiresAt sql.NullTime

		err := rows.Scan(
			&c.ID, &c.RequestID, &consentIDPtr, &c.UserID, &c.BankID, &requestingBank,
			&c.ConsentType, &amountField, &currencyField, &c.DebtorAccount, &creditorAccountField,
			&creditorName, &reference, &maxUses, &maxAmountPerPayment, &maxTotalAmount,
			pq.Array(&c.AllowedCreditorAccounts),
			&vrpMaxIndividualAmount, &vrpDailyLimit, &vrpMonthlyLimit,
			&validFrom, &validUntil, &reason, &c.Status, &expiresAt,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if consentIDPtr.Valid {
			c.ConsentID = &consentIDPtr.String
		}
		if requestingBank.Valid {
			c.RequestingBank = &requestingBank.String
		}
		if currencyField.Valid {
			c.Currency = &currencyField.String
		}
		if creditorAccountField.Valid {
			c.CreditorAccount = &creditorAccountField.String
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
		if amountField.Valid {
			val := amountField.Float64
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

		// Проверяем, подходит ли согласие для данного платежа
		matches := false
		if c.ConsentType == "single_use" {
			// Для single_use проверяем creditor_account и amount
			if c.CreditorAccount != nil && *c.CreditorAccount == creditorAccount {
				if c.Amount != nil && *c.Amount == amount {
					if c.Currency != nil && *c.Currency == currency {
						matches = true
					}
				}
			}
		} else if c.ConsentType == "multi_use" {
			// Для multi_use проверяем лимиты
			matches = true
			if c.MaxAmountPerPayment != nil && *c.MaxAmountPerPayment < amount {
				matches = false
			}
			if len(c.AllowedCreditorAccounts) > 0 {
				allowed := false
				for _, acc := range c.AllowedCreditorAccounts {
					if acc == creditorAccount {
						allowed = true
						break
					}
				}
				if !allowed {
					matches = false
				}
			}
			// Проверяем max_total_amount - получаем сумму всех платежей по этому согласию
			if c.MaxTotalAmount != nil {
				var totalSpent sql.NullFloat64
				var consentIDForQuery string
				if c.ConsentID != nil {
					consentIDForQuery = *c.ConsentID
				} else {
					consentIDForQuery = c.RequestID
				}
				err := r.DB.QueryRow(`
					SELECT COALESCE(SUM(amount), 0) 
					FROM payments 
					WHERE payment_consent_id = $1
				`, consentIDForQuery).Scan(&totalSpent)
				if err == nil && totalSpent.Valid {
					if *c.MaxTotalAmount < totalSpent.Float64+amount {
						matches = false
					}
				}
			}
			// Проверяем max_uses
			if c.MaxUses != nil {
				var usesCount sql.NullInt64
				var consentIDForQuery string
				if c.ConsentID != nil {
					consentIDForQuery = *c.ConsentID
				} else {
					consentIDForQuery = c.RequestID
				}
				err := r.DB.QueryRow(`
					SELECT COUNT(*) 
					FROM payments 
					WHERE payment_consent_id = $1
				`, consentIDForQuery).Scan(&usesCount)
				if err == nil && usesCount.Valid {
					if *c.MaxUses <= int(usesCount.Int64) {
						matches = false
					}
				}
			}
		} else if c.ConsentType == "vrp" {
			// Для vrp проверяем лимиты
			matches = true
			if c.VRPMaxIndividualAmount != nil && *c.VRPMaxIndividualAmount < amount {
				matches = false
			}
		}

		if matches {
			consents = append(consents, c)
		}
	}

	return consents, nil
}

// GetTotalAmountSpentForConsent получает общую сумму всех платежей по согласию
func (r *Repository) GetTotalAmountSpentForConsent(consentID string) (float64, error) {
	var total sql.NullFloat64
	err := r.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) 
		FROM payments 
		WHERE payment_consent_id = $1
	`, consentID).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Float64, nil
	}
	return 0, nil
}

// GetUsesCountForConsent получает количество использований согласия
func (r *Repository) GetUsesCountForConsent(consentID string) (int, error) {
	var count sql.NullInt64
	err := r.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM payments 
		WHERE payment_consent_id = $1
	`, consentID).Scan(&count)
	if err != nil {
		return 0, err
	}
	if count.Valid {
		return int(count.Int64), nil
	}
	return 0, nil
}

// UpdatePaymentConsentAfterPayment обновляет согласие после использования в платеже
func (r *Repository) UpdatePaymentConsentAfterPayment(consentID string, amount float64) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Получаем согласие
	var consentType string
	var maxUses, currentUses sql.NullInt64
	var maxTotalAmount, currentTotal sql.NullFloat64
	err = tx.QueryRow(`
		SELECT consent_type, max_uses, max_total_amount
		FROM payment_consents
		WHERE consent_id = $1 OR request_id = $1
	`, consentID).Scan(&consentType, &maxUses, &maxTotalAmount)
	if err != nil {
		return err
	}

	// Получаем текущее количество использований и общую сумму
	err = tx.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM payments
		WHERE payment_consent_id = $1
	`, consentID).Scan(&currentUses, &currentTotal)
	if err != nil {
		return err
	}

	if consentType == "single_use" {
		// Удаляем single_use согласие после использования
		_, err = tx.Exec(`
			DELETE FROM payment_consents
			WHERE consent_id = $1 OR request_id = $1
		`, consentID)
		if err != nil {
			return err
		}
	} else if consentType == "multi_use" {
		// Для multi_use ничего не обновляем здесь, так как max_total_amount и max_uses
		// проверяются при поиске согласий, а не хранятся как счетчики
		// Но можно добавить логику, если нужно отслеживать использование
	}

	return tx.Commit()
}

// DeletePaymentConsent удаляет согласие
func (r *Repository) DeletePaymentConsent(consentID string) error {
	_, err := r.DB.Exec(`
		DELETE FROM payment_consents
		WHERE consent_id = $1 OR request_id = $1
	`, consentID)
	return err
}
