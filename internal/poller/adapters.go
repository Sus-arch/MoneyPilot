package poller

import (
	"MoneyPilot/internal/storage"
	"log"
)

// --- AccountConsents adapter ---
type AccountConsentRepoAdapter struct {
	Repo *storage.Repository
}

func (a *AccountConsentRepoAdapter) GetPendingConsents() ([]ConsentRecord, error) {
	items, err := a.Repo.GetPendingAccountConsents()
	if err != nil {
		return nil, err
	}
	res := make([]ConsentRecord, 0, len(items))
	for _, c := range items {
		// Получаем client_id из user_id
		user, err := a.Repo.GetUserByID(c.UserID)
		clientID := ""
		if err == nil && user != nil {
			clientID = user.ClientID
		}
		res = append(res, ConsentRecord{
			ConsentID:      c.ConsentID,
			BankCode:       derefString(c.BankCode),
			UserID:         c.UserID,
			ClientID:       clientID,
			Status:         c.Status,
			ConsentType:    "account",
			RequestingBank: derefString(c.RequestingBank),
		})
	}
	return res, nil
}

func (a *AccountConsentRepoAdapter) UpdateConsentStatus(consentID string, status string) error {
	return a.Repo.UpdateAccountConsentStatusByConsentID(consentID, status)
}

func (a *AccountConsentRepoAdapter) UpdateConsentID(oldID, newID, status string) error {
	return a.Repo.UpdateAccountConsentIDAndStatus(oldID, newID, status)
}

// --- ProductConsents adapter ---
type ProductConsentRepoAdapter struct {
	Repo *storage.Repository
}

func (p *ProductConsentRepoAdapter) GetPendingConsents() ([]ConsentRecord, error) {
	rows, err := p.Repo.DB.Query(`
		SELECT 
			COALESCE(NULLIF(ac.consent_id, ''), ac.request_id) AS consent_id,
			b.code AS bank_code,
			ac.user_id,
			u.client_id,
			ac.requesting_bank,
			ac.status,
			ac.request_id
		FROM product_agreement_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		LEFT JOIN users u ON u.id = ac.user_id
		WHERE ac.status = 'pending' OR (ac.status = 'approved' AND (ac.consent_id IS NULL OR ac.consent_id = ''))
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ConsentRecord
	for rows.Next() {
		var c ConsentRecord
		var bankCode, reqBank *string
		var requestID string
		var clientID *string
		// Используем COALESCE для выбора consent_id, если он не пустой, иначе request_id
		if err := rows.Scan(&c.ConsentID, &bankCode, &c.UserID, &clientID, &reqBank, &c.Status, &requestID); err != nil {
			log.Printf("Error scanning product consent row: %v", err)
			return nil, err
		}
		// Если consent_id пустой, используем request_id
		if c.ConsentID == "" {
			c.ConsentID = requestID
		}
		c.BankCode = derefString(bankCode)
		c.ClientID = derefString(clientID)
		c.RequestingBank = derefString(reqBank)
		c.ConsentType = "product-agreement"
		results = append(results, c)
	}
	return results, nil
}

func (p *ProductConsentRepoAdapter) UpdateConsentStatus(consentID string, status string) error {
	// Обновляем по consent_id или request_id (если consent_id пустой)
	_, err := p.Repo.DB.Exec(`UPDATE product_agreement_consents SET status=$1 WHERE consent_id=$2 OR request_id=$2`, status, consentID)
	return err
}

func (p *ProductConsentRepoAdapter) UpdateConsentID(oldID, newID, status string) error {
	// Обновляем по consent_id или request_id (если consent_id пустой)
	// Если oldID - это request_id, то обновляем request_id, иначе consent_id
	_, err := p.Repo.DB.Exec(`
		UPDATE product_agreement_consents 
		SET consent_id=$1, status=$2 
		WHERE consent_id=$3 OR request_id=$3
	`, newID, status, oldID)
	return err
}

// --- PaymentConsents adapter ---
type PaymentConsentRepoAdapter struct {
	Repo *storage.Repository
}

func (p *PaymentConsentRepoAdapter) GetPendingConsents() ([]ConsentRecord, error) {
	rows, err := p.Repo.DB.Query(`
		SELECT 
			COALESCE(NULLIF(pc.consent_id, ''), pc.request_id) AS consent_id,
			b.code AS bank_code,
			pc.user_id,
			u.client_id,
			pc.requesting_bank,
			pc.status,
			pc.request_id
		FROM payment_consents pc
		LEFT JOIN banks b ON b.id = pc.bank_id
		LEFT JOIN users u ON u.id = pc.user_id
		WHERE pc.status = 'pending' OR (pc.status = 'approved' AND (pc.consent_id IS NULL OR pc.consent_id = ''))
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ConsentRecord
	for rows.Next() {
		var c ConsentRecord
		var bankCode, reqBank *string
		var requestID string
		var clientID *string
		// Используем COALESCE для выбора consent_id, если он не пустой, иначе request_id
		if err := rows.Scan(&c.ConsentID, &bankCode, &c.UserID, &clientID, &reqBank, &c.Status, &requestID); err != nil {
			log.Printf("Error scanning payment consent row: %v", err)
			return nil, err
		}
		// Если consent_id пустой, используем request_id
		if c.ConsentID == "" {
			c.ConsentID = requestID
		}
		c.BankCode = derefString(bankCode)
		c.ClientID = derefString(clientID)
		c.RequestingBank = derefString(reqBank)
		c.ConsentType = "payment"
		results = append(results, c)
	}
	return results, nil
}

func (p *PaymentConsentRepoAdapter) UpdateConsentStatus(consentID string, status string) error {
	// Обновляем по consent_id или request_id (если consent_id пустой)
	_, err := p.Repo.DB.Exec(`UPDATE payment_consents SET status=$1, updated_at=NOW() WHERE consent_id=$2 OR request_id=$2`, status, consentID)
	return err
}

func (p *PaymentConsentRepoAdapter) UpdateConsentID(oldID, newID, status string) error {
	// Обновляем по consent_id или request_id (если consent_id пустой)
	_, err := p.Repo.DB.Exec(`
		UPDATE payment_consents 
		SET consent_id=$1, status=$2, updated_at=NOW() 
		WHERE consent_id=$3 OR request_id=$3
	`, newID, status, oldID)
	return err
}

// helper
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
