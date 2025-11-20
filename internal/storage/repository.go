package storage

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/lib/pq"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// Deprecated: SaveConsent kept for compatibility but project uses account_consents table now.
func (r *Repository) SaveConsent(userID, bankID int, consent_id string, expires time.Time, requesting_bank string) error {
	_, err := r.DB.Exec(
		`INSERT INTO account_consents (consent_id, user_id, bank_id, requesting_bank, permissions, status, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		consent_id, userID, bankID, requesting_bank, nil, "approved", expires,
	)
	return err
}

func (r *Repository) GetBankByCode(code string) (*Bank, error) {
	var b Bank
	err := r.DB.QueryRow(`SELECT id, code, name, api_base_url, jwks_url, created_at FROM banks WHERE code=$1`, code).
		Scan(&b.ID, &b.Code, &b.Name, &b.APIBase, &b.JWKSURL, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// GetBankByID получает банк по ID
func (r *Repository) GetBankByID(id int) (*Bank, error) {
	var b Bank
	err := r.DB.QueryRow(`SELECT id, code, name, api_base_url, jwks_url, created_at FROM banks WHERE id=$1`, id).
		Scan(&b.ID, &b.Code, &b.Name, &b.APIBase, &b.JWKSURL, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) GetUserByID(id int) (*User, error) {
	var b User
	err := r.DB.QueryRow(`SELECT id, client_id, bank_id, email, password_hash, segment, created_at FROM users WHERE id=$1`, id).
		Scan(&b.ID, &b.ClientID, &b.BankID, &b.Email, &b.PasswordHash, &b.Segment, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// GetUserByClientIDAndBank returns a user by client_id and optional bank code.
// If bankCode is empty, it returns the user matching client_id regardless of bank.
// If bankCode is provided, it returns a user whose client_id matches and whose bank_id is either NULL (global) or matches the bank's id.
func (r *Repository) GetUserByClientIDAndBank(clientID, bankCode string) (*User, error) {
	var u User
	if bankCode == "" {
		err := r.DB.QueryRow(`SELECT id, client_id, bank_id, email, password_hash, segment, created_at FROM users WHERE client_id=$1`, clientID).
			Scan(&u.ID, &u.ClientID, &u.BankID, &u.Email, &u.PasswordHash, &u.Segment, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		return &u, nil
	}

	// resolve bank id
	var bankID int
	if err := r.DB.QueryRow(`SELECT id FROM banks WHERE code=$1`, bankCode).Scan(&bankID); err != nil {
		return nil, err
	}

	// allow user records that are global (bank_id IS NULL) or tied to the specific bank
	err := r.DB.QueryRow(`SELECT id, client_id, bank_id, email, password_hash, segment, created_at FROM users WHERE client_id=$1 AND (bank_id IS NULL OR bank_id=$2)`, clientID, bankID).
		Scan(&u.ID, &u.ClientID, &u.BankID, &u.Email, &u.PasswordHash, &u.Segment, &u.CreatedAt)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return &u, nil
}

// GetUserByClientIDAndBankID возвращает пользователя по client_id и bank_id
// Используется как в SaveProductAgreementConsent для правильного получения userID для конкретного банка
func (r *Repository) GetUserByClientIDAndBankID(clientID string, bankID int) (*User, error) {
	var u User
	err := r.DB.QueryRow(`SELECT id, client_id, bank_id, email, password_hash, segment, created_at FROM users WHERE client_id=$1 AND bank_id=$2`, clientID, bankID).
		Scan(&u.ID, &u.ClientID, &u.BankID, &u.Email, &u.PasswordHash, &u.Segment, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SaveAccountConsentByEmailAndBank inserts an account_consents row by resolving user and bank
// from human-friendly values (email and bank code). This avoids requiring caller to know DB ids.
// Использует client_id и bank_id для получения правильного userID, как в SaveProductAgreementConsent
func (r *Repository) SaveAccountConsentByClientIdAndBank(client_id, bankCode, consentID, requestingBank string, permissions []string, status string, expires time.Time) error {
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

	// Получаем правильного пользователя для данного банка по client_id и bank_id
	// Как в SaveProductAgreementConsent: SELECT id FROM users WHERE client_id=$1 AND bank_id=$2
	var userID int
	if err := tx.QueryRow(`SELECT id FROM users WHERE client_id=$1 AND bank_id=$2`, client_id, bankID).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found for this bank")
		}
		return err
	}

	_, err = tx.Exec(`INSERT INTO account_consents (consent_id, user_id, bank_id, requesting_bank, permissions, status, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, consentID, userID, bankID, requestingBank, pq.Array(permissions), status, expires)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetValidAccountConsentsByEmail(email string) ([]AccountConsent, error) {
	var userID int
	if err := r.DB.QueryRow(`SELECT id FROM users WHERE email=$1`, email).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	rows, err := r.DB.Query(`
		SELECT ac.id, ac.consent_id, ac.user_id, ac.bank_id, b.code AS bank_code, ac.requesting_bank, ac.permissions, ac.status, ac.expires_at, ac.created_at
		FROM account_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.user_id=$1 AND (ac.expires_at IS NULL OR ac.expires_at > NOW())
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []AccountConsent
	for rows.Next() {
		var c AccountConsent
		if err := rows.Scan(&c.ID, &c.ConsentID, &c.UserID, &c.BankID, &c.BankCode, &c.RequestingBank, pq.Array(&c.Permissions), &c.Status, &c.ExpiresAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return consents, nil
}

func (r *Repository) GetValidAccountConsentsByEmailAndBank(email, bank string) ([]AccountConsent, error) {
	var userID int
	if err := r.DB.QueryRow(`SELECT id FROM users WHERE email=$1`, email).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	var BankID int
	if err := r.DB.QueryRow(`SELECT id FROM banks WHERE code=$1`, bank).Scan(&BankID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("bank not found")
		}
		return nil, err
	}
	rows, err := r.DB.Query(`
		SELECT ac.id, ac.consent_id, ac.user_id, ac.bank_id, b.code AS bank_code, ac.requesting_bank, ac.permissions, ac.status, ac.expires_at, ac.created_at
		FROM account_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.user_id=$1 AND b.code=$2 AND (ac.expires_at IS NULL OR ac.expires_at > NOW())
	`, userID, BankID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []AccountConsent
	for rows.Next() {
		var c AccountConsent
		if err := rows.Scan(&c.ID, &c.ConsentID, &c.UserID, &c.BankID, &c.BankCode, &c.RequestingBank, pq.Array(&c.Permissions), &c.Status, &c.ExpiresAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return consents, nil
}

func (r *Repository) GetValidAccountConsentsByUserIDAndBank(userID int, bankcode string) ([]AccountConsent, error) {
	user, _ := r.GetUserByID(userID)
	rows, err := r.DB.Query(`
		SELECT 
			ac.id,
			ac.consent_id,
			ac.user_id,
			ac.bank_id,
			b.code AS bank_code,
			ac.requesting_bank,
			ac.permissions,
			ac.status,
			ac.expires_at,
			ac.created_at
		FROM account_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.user_id IN (SELECT id FROM users WHERE client_id=$1) AND b.code=$2
		  AND (ac.expires_at IS NULL OR ac.expires_at > NOW())
	`, user.ClientID, bankcode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []AccountConsent
	for rows.Next() {
		var c AccountConsent
		if err := rows.Scan(
			&c.ID,
			&c.ConsentID,
			&c.UserID,
			&c.BankID,
			&c.BankCode,
			&c.RequestingBank,
			pq.Array(&c.Permissions),
			&c.Status,
			&c.ExpiresAt,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}

	return consents, nil
}

func (r *Repository) GetValidAccountConsentsByUserID(userID int) ([]AccountConsent, error) {
	user, _ := r.GetUserByID(userID)
	rows, err := r.DB.Query(`
		SELECT 
			ac.id,
			ac.consent_id,
			ac.user_id,
			ac.bank_id,
			b.code AS bank_code,
			ac.requesting_bank,
			ac.permissions,
			ac.status,
			ac.expires_at,
			ac.created_at
		FROM account_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.user_id IN (SELECT id FROM users WHERE client_id=$1)
		  AND (ac.expires_at IS NULL OR ac.expires_at > NOW())
	`, user.ClientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []AccountConsent
	for rows.Next() {
		var c AccountConsent
		if err := rows.Scan(
			&c.ID,
			&c.ConsentID,
			&c.UserID,
			&c.BankID,
			&c.BankCode,
			&c.RequestingBank,
			pq.Array(&c.Permissions),
			&c.Status,
			&c.ExpiresAt,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}

	return consents, nil
}

// GetPendingAccountConsents returns all consents in DB that are currently marked as 'pending'.
// It includes the bank code (joined from banks table) so callers can route requests to the correct bank.
func (r *Repository) GetPendingAccountConsents() ([]AccountConsent, error) {

	rows, err := r.DB.Query(`
		SELECT ac.id, ac.consent_id, ac.user_id, ac.bank_id, b.code AS bank_code, ac.requesting_bank, ac.permissions, ac.status, ac.expires_at, ac.created_at
		FROM account_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.status = $1
	`, "pending")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []AccountConsent
	for rows.Next() {
		var c AccountConsent
		if err := rows.Scan(&c.ID, &c.ConsentID, &c.UserID, &c.BankID, &c.BankCode, &c.RequestingBank, pq.Array(&c.Permissions), &c.Status, &c.ExpiresAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return consents, nil
}

// UpdateAccountConsentStatusByConsentID updates the status of an account_consent row identified by consent_id.
func (r *Repository) UpdateAccountConsentStatusByConsentID(consentID string, status string) error {
	_, err := r.DB.Exec(`UPDATE account_consents SET status=$1 WHERE consent_id=$2`, status, consentID)
	return err
}

// UpdateAccountConsentIDAndStatus replaces the consent_id for a row and updates its status.
// This is used when a bank first returned a temporary request id and later provides a final consent id.
func (r *Repository) UpdateAccountConsentIDAndStatus(oldConsentID, newConsentID, status string) error {
	_, err := r.DB.Exec(`UPDATE account_consents SET consent_id=$1, status=$2 WHERE consent_id=$3`, newConsentID, status, oldConsentID)
	return err
}

func (r *Repository) SaveProductAgreementConsent(clientID, bankCode, requestID, consentID, requestingBank string,
	read, open, close bool, allowedTypes []string, maxAmount float64, status string, expiresAt time.Time) error {

	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID, bankID int

	if err := tx.QueryRow(`SELECT id FROM banks WHERE code=$1`, bankCode).Scan(&bankID); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT id FROM users WHERE client_id=$1 AND bank_id=$2`, clientID, bankID).Scan(&userID); err != nil {
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO product_agreement_consents (
			request_id, consent_id, user_id, bank_id, requesting_bank,
			read_product_agreements, open_product_agreements, close_product_agreements,
			allowed_product_types, max_amount, status, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, requestID, consentID, userID, bankID, requestingBank, read, open, close, pq.Array(allowedTypes), maxAmount, status, expiresAt)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetUserByUserIDAndBank(userID int, bankCode string) (*User, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return nil, err
	}
	var bankID int

	if err := tx.QueryRow(`SELECT id FROM banks WHERE code=$1`, bankCode).Scan(&bankID); err != nil {
		return nil, err
	}

	var clientId string
	if err := tx.QueryRow(`SELECT client_id FROM users WHERE id=$1`, userID).Scan(&clientId); err != nil {
		return nil, err
	}
	var u User

	err = r.DB.QueryRow(`SELECT id, client_id,bank_id, email, password_hash,segment,created_at  FROM users WHERE client_id=$1 AND bank_id=$2`, clientId, bankID).
		Scan(&u.ID, &u.ClientID, &u.BankID, &u.Email, &u.PasswordHash, &u.Segment, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil

}

func (r *Repository) GetActiveProductConsentByUserAndBank(userID int, bankCode string) (*ProductAgreementConsent, error) {
	user, err := r.GetUserByUserIDAndBank(userID, bankCode)
	log.Println(user)
	rows, err := r.DB.Query(`
		SELECT ac.id, ac.request_id , ac.consent_id, ac.user_id, ac.bank_id, b.code, ac.requesting_bank,
			   ac.read_product_agreements, ac.open_product_agreements, ac.close_product_agreements,
			   ac.allowed_product_types, ac.max_amount, ac.status, ac.expires_at, ac.created_at
		FROM product_agreement_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.user_id=$1 AND b.code=$2 AND ac.status IN ('approved','pending')
	`, user.ID, bankCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var c ProductAgreementConsent
		if err := rows.Scan(&c.ID, &c.RequestID, &c.ConsentID, &c.UserID, &c.BankID, &c.BankCode,
			&c.RequestingBank, &c.ReadProductAgreements, &c.OpenProductAgreements, &c.CloseProductAgreements,
			pq.Array(&c.AllowedProductTypes), &c.MaxAmount, &c.Status, &c.ExpiresAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		return &c, nil
	}
	return nil, sql.ErrNoRows
}

func (r *Repository) DeleteProductConsent(consentID string) error {
	_, err := r.DB.Exec(`DELETE FROM product_agreement_consents WHERE consent_id=$1`, consentID)
	return err
}

// UpsertProductAgreement сохраняет или обновляет договор по продукту в БД
func (r *Repository) UpsertProductAgreement(userID int, agreementID, productID string, amount *float64, termMonths *int, status string) error {
	_, err := r.DB.Exec(`
		INSERT INTO product_agreements (agreement_id, user_id, product_id, amount, term_months, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (agreement_id)
		DO UPDATE SET
			product_id = EXCLUDED.product_id,
			amount = EXCLUDED.amount,
			term_months = EXCLUDED.term_months,
			status = EXCLUDED.status,
			user_id = EXCLUDED.user_id
	`, agreementID, userID, productID, amount, termMonths, status)
	return err
}

// UpsertProduct сохраняет или обновляет продукт каталога в БД
func (r *Repository) UpsertProduct(bankID int, productID, productType, productName string, description *string, interestRate, minAmount, maxAmount *float64, termMonths *int) error {
	_, err := r.DB.Exec(`
		INSERT INTO products (product_id, bank_id, product_type, name, description, interest_rate, min_amount, max_amount, term_months, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (product_id)
		DO UPDATE SET
			bank_id = EXCLUDED.bank_id,
			product_type = EXCLUDED.product_type,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			interest_rate = EXCLUDED.interest_rate,
			min_amount = EXCLUDED.min_amount,
			max_amount = EXCLUDED.max_amount,
			term_months = EXCLUDED.term_months,
			updated_at = NOW()
	`, productID, bankID, productType, productName, description, interestRate, minAmount, maxAmount, termMonths)
	return err
}

// UpsertAccount сохраняет или обновляет счет в БД
// Использует составной уникальный ключ (user_id, bank_id, account_number)
func (r *Repository) UpsertAccount(
	userID, bankID int,
	accountNumber, accountType, accountSubType, nickname, currency, status string,
	ownerName, schemeName, identification *string,
	openingDate *time.Time,
) error {
	_, err := r.DB.Exec(`
		INSERT INTO accounts (
			user_id, bank_id, account_number, account_type, account_subtype,
			nickname, currency, status, owner_name, opening_date,
			scheme_name, identification, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (user_id, bank_id, account_number) 
		DO UPDATE SET 
			account_type = EXCLUDED.account_type,
			account_subtype = EXCLUDED.account_subtype,
			nickname = EXCLUDED.nickname,
			currency = EXCLUDED.currency,
			status = EXCLUDED.status,
			owner_name = EXCLUDED.owner_name,
			opening_date = EXCLUDED.opening_date,
			scheme_name = EXCLUDED.scheme_name,
			identification = EXCLUDED.identification,
			updated_at = NOW()
	`, userID, bankID, accountNumber, accountType, accountSubType,
		nickname, currency, status, ownerName, openingDate,
		schemeName, identification)
	return err
}

// UpdateAccountBalance обновляет баланс счета
func (r *Repository) UpdateAccountBalance(userID, bankID int, accountNumber string, balance float64) error {
	_, err := r.DB.Exec(`
		UPDATE accounts 
		SET balance = $1 
		WHERE user_id = $2 AND bank_id = $3 AND account_number = $4
	`, balance, userID, bankID, accountNumber)
	return err
}

// GetAccountByNumber получает счет по номеру
func (r *Repository) GetAccountByNumber(accountNumber string) (*Account, error) {
	var acc Account
	var accountSubType, nickname, ownerName, schemeName, identification sql.NullString
	var openingDate sql.NullTime

	err := r.DB.QueryRow(`
		SELECT id, user_id, bank_id, account_number, account_type, account_subtype,
			nickname, currency, balance, status, owner_name, opening_date,
			scheme_name, identification, created_at, updated_at
		FROM accounts 
		WHERE account_number = $1
	`, accountNumber).Scan(
		&acc.ID, &acc.UserID, &acc.BankID, &acc.AccountNumber, &acc.AccountType,
		&accountSubType, &nickname, &acc.Currency, &acc.Balance, &acc.Status,
		&ownerName, &openingDate, &schemeName, &identification,
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
	if identification.Valid {
		acc.Identification = &identification.String
	}

	return &acc, nil
}

// GetAccountsByUserID получает все счета пользователя
func (r *Repository) GetAccountsByUserID(userID int) ([]Account, error) {
	rows, err := r.DB.Query(`
		SELECT id, user_id, bank_id, account_number, account_type, account_subtype,
			nickname, currency, balance, status, owner_name, opening_date,
			scheme_name, identification, created_at, updated_at
		FROM accounts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var acc Account
		var accountSubType, nickname, ownerName, schemeName, identification sql.NullString
		var openingDate sql.NullTime

		err := rows.Scan(
			&acc.ID, &acc.UserID, &acc.BankID, &acc.AccountNumber, &acc.AccountType,
			&accountSubType, &nickname, &acc.Currency, &acc.Balance, &acc.Status,
			&ownerName, &openingDate, &schemeName, &identification,
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
		if identification.Valid {
			acc.Identification = &identification.String
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// GetAccountsByClientID получает все счета для всех пользователей с одинаковым client_id
func (r *Repository) GetAccountsByClientID(clientID string) ([]Account, error) {
	rows, err := r.DB.Query(`
		SELECT a.id, a.user_id, a.bank_id, a.account_number, a.account_type, a.account_subtype,
			a.nickname, a.currency, a.balance, a.status, a.owner_name, a.opening_date,
			a.scheme_name, a.identification, a.created_at, a.updated_at
		FROM accounts a
		INNER JOIN users u ON u.id = a.user_id
		WHERE u.client_id = $1
		ORDER BY a.created_at DESC
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var acc Account
		var accountSubType, nickname, ownerName, schemeName, identification sql.NullString
		var openingDate sql.NullTime

		err := rows.Scan(
			&acc.ID, &acc.UserID, &acc.BankID, &acc.AccountNumber, &acc.AccountType,
			&accountSubType, &nickname, &acc.Currency, &acc.Balance, &acc.Status,
			&ownerName, &openingDate, &schemeName, &identification,
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
		if identification.Valid {
			acc.Identification = &identification.String
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// GetValidPaymentConsentsByUserIDAndBank получает активные согласия на платежи для пользователя и банка
func (r *Repository) GetValidPaymentConsentsByUserIDAndBank(userID int, bankCode, consentType string) ([]PaymentConsent, error) {
	user, err := r.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	var query string
	var args []interface{}

	if consentType != "" {
		query = `
			SELECT 
				pc.id, pc.request_id, pc.consent_id, pc.user_id, pc.bank_id, 
				b.code AS bank_code, pc.requesting_bank, pc.consent_type,
				pc.amount, pc.currency, pc.debtor_account, pc.creditor_account,
				pc.creditor_name, pc.reference, pc.max_uses, pc.max_amount_per_payment,
				pc.max_total_amount, pc.allowed_creditor_accounts,
				pc.vrp_max_individual_amount, pc.vrp_daily_limit, pc.vrp_monthly_limit,
				pc.valid_from, pc.valid_until, pc.reason, pc.status, pc.expires_at,
				pc.created_at, pc.updated_at
			FROM payment_consents pc
			LEFT JOIN banks b ON b.id = pc.bank_id
			WHERE pc.user_id IN (SELECT id FROM users WHERE client_id=$1)
			  AND b.code = $2
			  AND pc.consent_type = $3
			  AND pc.status IN ('approved', 'pending')
			  AND (pc.valid_until IS NULL OR pc.valid_until > NOW())
		`
		args = []interface{}{user.ClientID, bankCode, consentType}
	} else {
		query = `
			SELECT 
				pc.id, pc.request_id, pc.consent_id, pc.user_id, pc.bank_id, 
				b.code AS bank_code, pc.requesting_bank, pc.consent_type,
				pc.amount, pc.currency, pc.debtor_account, pc.creditor_account,
				pc.creditor_name, pc.reference, pc.max_uses, pc.max_amount_per_payment,
				pc.max_total_amount, pc.allowed_creditor_accounts,
				pc.vrp_max_individual_amount, pc.vrp_daily_limit, pc.vrp_monthly_limit,
				pc.valid_from, pc.valid_until, pc.reason, pc.status, pc.expires_at,
				pc.created_at, pc.updated_at
			FROM payment_consents pc
			LEFT JOIN banks b ON b.id = pc.bank_id
			WHERE pc.user_id IN (SELECT id FROM users WHERE client_id=$1)
			  AND b.code = $2
			  AND pc.status IN ('approved', 'pending')
			  AND (pc.valid_until IS NULL OR pc.valid_until > NOW())
		`
		args = []interface{}{user.ClientID, bankCode}
	}

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []PaymentConsent
	for rows.Next() {
		var c PaymentConsent
		var consentID, requestingBank, currency, creditorAccount, creditorName, reference, reason sql.NullString
		var amount, maxAmountPerPayment, maxTotalAmount, vrpMaxIndividualAmount, vrpDailyLimit, vrpMonthlyLimit sql.NullFloat64
		var maxUses sql.NullInt64
		var validFrom, validUntil, expiresAt sql.NullTime
		var bankCode *string

		err := rows.Scan(
			&c.ID, &c.RequestID, &consentID, &c.UserID, &c.BankID, &bankCode, &requestingBank,
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

		if consentID.Valid {
			c.ConsentID = &consentID.String
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
		c.BankCode = bankCode

		consents = append(consents, c)
	}

	return consents, nil
}

// SavePaymentConsentByClientIdAndBank сохраняет согласие на платеж
func (r *Repository) SavePaymentConsentByClientIdAndBank(
	clientID, bankCode, requestID, consentID, requestingBank, consentType, currency, debtorAccount string,
	creditorAccount, creditorName, reference *string,
	amount, maxAmountPerPayment, maxTotalAmount, vrpMaxIndividualAmount, vrpDailyLimit, vrpMonthlyLimit *float64,
	maxUses *int,
	allowedCreditorAccounts []string,
	validFrom, validUntil *time.Time,
	reason *string,
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

	// Получаем правильного пользователя для данного банка по client_id и bank_id
	var userID int
	if err := tx.QueryRow(`SELECT id FROM users WHERE client_id=$1 AND bank_id=$2`, clientID, bankID).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found for this bank")
		}
		return err
	}

	var consentIDPtr interface{}
	if consentID != "" {
		consentIDPtr = consentID
	} else {
		consentIDPtr = nil
	}

	_, err = tx.Exec(`
		INSERT INTO payment_consents (
			request_id, consent_id, user_id, bank_id, requesting_bank, consent_type,
			amount, currency, debtor_account, creditor_account, creditor_name, reference,
			max_uses, max_amount_per_payment, max_total_amount, allowed_creditor_accounts,
			vrp_max_individual_amount, vrp_daily_limit, vrp_monthly_limit,
			valid_from, valid_until, reason, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`,
		requestID, consentIDPtr, userID, bankID, requestingBank, consentType,
		amount, currency, debtorAccount, creditorAccount, creditorName, reference,
		maxUses, maxAmountPerPayment, maxTotalAmount, pq.Array(allowedCreditorAccounts),
		vrpMaxIndividualAmount, vrpDailyLimit, vrpMonthlyLimit,
		validFrom, validUntil, reason, status)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdatePaymentConsentStatusByRequestID обновляет статус согласия на платеж по request_id
func (r *Repository) UpdatePaymentConsentStatusByRequestID(requestID, status string) error {
	_, err := r.DB.Exec(`UPDATE payment_consents SET status=$1, updated_at=NOW() WHERE request_id=$2`, status, requestID)
	return err
}

// UpdatePaymentConsentIDAndStatus обновляет consent_id и статус согласия на платеж
func (r *Repository) UpdatePaymentConsentIDAndStatus(requestID, consentID, status string) error {
	_, err := r.DB.Exec(`
		UPDATE payment_consents 
		SET consent_id=$1, status=$2, updated_at=NOW() 
		WHERE request_id=$3
	`, consentID, status, requestID)
	return err
}

// GetPendingPaymentConsents получает все согласия на платежи со статусом pending
func (r *Repository) GetPendingPaymentConsents() ([]PaymentConsent, error) {
	rows, err := r.DB.Query(`
		SELECT 
			pc.id, pc.request_id, pc.consent_id, pc.user_id, pc.bank_id, 
			b.code AS bank_code, pc.requesting_bank, pc.consent_type,
			pc.amount, pc.currency, pc.debtor_account, pc.creditor_account,
			pc.creditor_name, pc.reference, pc.max_uses, pc.max_amount_per_payment,
			pc.max_total_amount, pc.allowed_creditor_accounts,
			pc.vrp_max_individual_amount, pc.vrp_daily_limit, pc.vrp_monthly_limit,
			pc.valid_from, pc.valid_until, pc.reason, pc.status, pc.expires_at,
			pc.created_at, pc.updated_at
		FROM payment_consents pc
		LEFT JOIN banks b ON b.id = pc.bank_id
		WHERE pc.status = 'pending' OR (pc.status = 'approved' AND (pc.consent_id IS NULL OR pc.consent_id = ''))
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []PaymentConsent
	for rows.Next() {
		var c PaymentConsent
		var consentID, requestingBank, currency, creditorAccount, creditorName, reference, reason sql.NullString
		var amount, maxAmountPerPayment, maxTotalAmount, vrpMaxIndividualAmount, vrpDailyLimit, vrpMonthlyLimit sql.NullFloat64
		var maxUses sql.NullInt64
		var validFrom, validUntil, expiresAt sql.NullTime
		var bankCode *string

		err := rows.Scan(
			&c.ID, &c.RequestID, &consentID, &c.UserID, &c.BankID, &bankCode, &requestingBank,
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

		if consentID.Valid {
			c.ConsentID = &consentID.String
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
		c.BankCode = bankCode

		consents = append(consents, c)
	}

	return consents, nil
}
