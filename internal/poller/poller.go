package poller

import (
	"MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/websockets"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ConsentRecord struct {
	ConsentID      string
	BankCode       string
	UserID         int
	ClientID       string // client_id для запросов к API
	Status         string
	ConsentType    string // "account" | "product"
	RequestingBank string
}

// Интерфейс, который должен реализовать любой репозиторий с согласиями (на счета, продукты и т.д.)
type ConsentRepo interface {
	GetPendingConsents() ([]ConsentRecord, error)
	UpdateConsentStatus(consentID string, status string) error
	UpdateConsentID(oldID, newID, status string) error
}

// Poller — универсальный менеджер согласий
type Poller struct {
	Repos       []ConsentRepo
	BankClients map[string]*bankapi.BankClient
	TokenSvc    *bankapi.TokenService
	HTTPClient  *http.Client

	subscribers map[string]chan string // consentID → канал, который ждёт подтверждения
	mu          sync.Mutex
	WSHub       *websockets.WebSocketHub
}

// Создаёт новый Poller
func NewPoller(
	repos []ConsentRepo,
	tokenSvc *bankapi.TokenService,
	banks map[string]*bankapi.BankClient,
	hub *websockets.WebSocketHub, // новый аргумент

) *Poller {
	return &Poller{
		Repos:       repos,
		TokenSvc:    tokenSvc,
		BankClients: banks,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
		subscribers: make(map[string]chan string),
		WSHub:       hub,
	}
}

// 🔁 Запуск фонового процесса опроса
func (p *Poller) Start(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	log.Printf("[poller] started, interval=%s", interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p.pollAll()
			case <-stopCh:
				log.Println("[poller] stopping")
				return
			}
		}
	}()
}

// Проверка всех согласий из всех репозиториев
func (p *Poller) pollAll() {
	for _, repo := range p.Repos {
		consents, err := repo.GetPendingConsents()
		if err != nil {
			log.Printf("[poller] failed to load consents: %v", err)
			continue
		}
		for _, c := range consents {
			go p.checkConsentStatus(repo, c)
		}
	}
}

// Проверка конкретного согласия через API банка
func (p *Poller) checkConsentStatus(repo ConsentRepo, c ConsentRecord) {
	if c.BankCode == "" {
		return
	}
	client := p.BankClients[c.BankCode]
	if client == nil {
		log.Printf("[poller] unknown bank code %s", c.BankCode)
		return
	}

	token, err := p.TokenSvc.GetValidToken(client)
	if err != nil {
		log.Printf("[poller] failed to get token for %s: %v", c.BankCode, err)
		return
	}

	// endpoint зависит от типа согласия
	var url string
	if c.ConsentType == "product-agreement" {
		// Для product-agreement путь: /product-agreement-consents/consent/{consentID}?client_id={clientID}
		url = fmt.Sprintf("%s/product-agreement-consents/consent/%s?client_id=%s",
			strings.TrimRight(client.BaseURL, "/"), c.ConsentID, c.ClientID)
	} else if c.ConsentType == "payment" {
		// Для payment путь: /payment-consents/{consentID}?client_id={clientID}
		url = fmt.Sprintf("%s/payment-consents/%s?client_id=%s",
			strings.TrimRight(client.BaseURL, "/"), c.ConsentID, c.ClientID)
	} else {
		// Для account путь: /account-consents/{consentID}
		url = fmt.Sprintf("%s/%s-consents/%s", strings.TrimRight(client.BaseURL, "/"), c.ConsentType, c.ConsentID)
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		log.Printf("[poller] request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[poller] non-200 for %s: %d %s", c.ConsentID, resp.StatusCode, string(body))
		return
	}

	var status string
	var consentID string

	if c.ConsentType == "payment" {
		// Для payment ответ не обернут в {"data": {...}}, поля на верхнем уровне
		var paymentResp struct {
			RequestID string `json:"request_id"`
			ConsentID string `json:"consent_id"`
			Status    string `json:"status"`
		}
		if err := json.Unmarshal(body, &paymentResp); err != nil {
			log.Printf("[poller] unmarshal failed for payment: %v (body=%s)", err, string(body))
			return
		}
		status = paymentResp.Status
		// Используем consent_id только если он пришел из API (не пустой)
		consentID = paymentResp.ConsentID
		log.Printf("[poller] parsed payment request_id=%s, consent_id=%s, status=%s",
			paymentResp.RequestID, consentID, status)
	} else if c.ConsentType == "product-agreement" {
		// Для product-agreement ответ не обернут в {"data": {...}}, поля на верхнем уровне
		var productResp struct {
			RequestID string `json:"request_id"`
			ConsentID string `json:"consent_id"`
			Status    string `json:"status"`
		}
		if err := json.Unmarshal(body, &productResp); err != nil {
			log.Printf("[poller] unmarshal failed for product-agreement: %v (body=%s)", err, string(body))
			return
		}
		status = productResp.Status
		// Используем consent_id только если он пришел из API (не пустой)
		// Если consent_id пустой, оставляем пустым (не используем request_id)
		consentID = productResp.ConsentID
		log.Printf("[poller] parsed product-agreement request_id=%s, consent_id=%s, status=%s",
			productResp.RequestID, consentID, status)
	} else {
		// Для account-consents ответ обернут в {"data": {...}}
		var wrapper struct {
			Data struct {
				ConsentID string `json:"consentId"`
				Status    string `json:"status"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &wrapper); err != nil {
			log.Printf("[poller] unmarshal failed: %v (body=%s)", err, string(body))
			return
		}
		status = wrapper.Data.Status
		consentID = wrapper.Data.ConsentID
		log.Printf("[poller] parsed account consentId=%s, status=%s", consentID, status)
	}

	if strings.EqualFold(status, "approved") ||
		strings.EqualFold(status, "authorized") ||
		strings.EqualFold(status, "authorised") {

		log.Printf("[poller] consent %s approved ✅", c.ConsentID)
		repo.UpdateConsentStatus(c.ConsentID, "approved")
		// Для product-agreement и payment: если пришел consent_id из API (не пустой), обновляем в БД
		// c.ConsentID в этом случае это request_id (из GetPendingConsents мы подставляем request_id если consent_id пустой)
		if (c.ConsentType == "product-agreement" || c.ConsentType == "payment") && consentID != "" {
			// Обновляем consent_id в БД по request_id
			log.Printf("[poller] updating consent_id for %s: request_id=%s, new_consent_id=%s",
				c.ConsentType, c.ConsentID, consentID)
			if err := repo.UpdateConsentID(c.ConsentID, consentID, "approved"); err != nil {
				log.Printf("[poller] failed to update consent_id: %v", err)
			} else {
				log.Printf("[poller] consent_id updated successfully ✅")
			}
		} else if c.ConsentType == "account" && consentID != "" && consentID != c.ConsentID {
			// Для account-consents: обновляем только если consent_id отличается
			repo.UpdateConsentID(c.ConsentID, consentID, "approved")
		}
		p.notifySubscribers(c.ConsentID)
	}
}

// 🔔 Подписка на уведомление о согласии
func (p *Poller) WaitForConsent(consentID string, timeout time.Duration) bool {
	ch := make(chan string, 1)
	p.mu.Lock()
	p.subscribers[consentID] = ch
	p.mu.Unlock()

	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (p *Poller) notifySubscribers(consentID string) {
	p.mu.Lock()
	if ch, ok := p.subscribers[consentID]; ok {
		ch <- "approved"
		close(ch)
		delete(p.subscribers, consentID)
	}
	p.mu.Unlock()

	// 🔔 Шлём уведомление фронту через WebSocket
	if p.WSHub != nil {
		msg := fmt.Sprintf(`{"consent_id":"%s","status":"approved"}`, consentID)
		p.WSHub.Broadcast(msg)
		log.Printf("[poller→ws] sent WS update: %s", msg)
	}
}
