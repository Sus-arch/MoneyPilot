package poller

import (
	"MoneyPilot/internal/storage"
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
		res = append(res, ConsentRecord{
			ConsentID:      c.ConsentID,
			BankCode:       derefString(c.BankCode),
			UserID:         c.UserID,
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
		SELECT ac.id, ac.consent_id, ac.user_id, b.code AS bank_code, ac.requesting_bank, ac.status
		FROM product_agreement_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.status = 'pending'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ConsentRecord
	for rows.Next() {
		var c ConsentRecord
		var bankCode, reqBank string
		if err := rows.Scan(&c.UserID, &c.ConsentID, &bankCode, &reqBank, &c.Status); err != nil {
			return nil, err
		}
		c.BankCode = bankCode
		c.RequestingBank = reqBank
		c.ConsentType = "product-agreement"
		results = append(results, c)
	}
	return results, nil
}

func (p *ProductConsentRepoAdapter) UpdateConsentStatus(consentID string, status string) error {
	_, err := p.Repo.DB.Exec(`UPDATE product_agreement_consents SET status=$1 WHERE consent_id=$2`, status, consentID)
	return err
}

func (p *ProductConsentRepoAdapter) UpdateConsentID(oldID, newID, status string) error {
	_, err := p.Repo.DB.Exec(`UPDATE product_agreement_consents SET consent_id=$1, status=$2 WHERE consent_id=$3`, newID, status, oldID)
	return err
}

// helper
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
