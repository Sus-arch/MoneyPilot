package consentpoller

import (
	"MoneyPilot/internal/bankapi"
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
}

// Создаёт новый Poller
func NewPoller(
	repos []ConsentRepo,
	tokenSvc *bankapi.TokenService,
	banks map[string]*bankapi.BankClient,
) *Poller {
	return &Poller{
		Repos:       repos,
		TokenSvc:    tokenSvc,
		BankClients: banks,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
		subscribers: make(map[string]chan string),
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
	url := fmt.Sprintf("%s/%s-consents/%s", strings.TrimRight(client.BaseURL, "/"), c.ConsentType, c.ConsentID)

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

	var parsed struct {
		Status    string `json:"status"`
		ConsentID string `json:"consent_id"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.Printf("[poller] unmarshal failed: %v", err)
		return
	}

	if strings.EqualFold(parsed.Status, "approved") ||
		strings.EqualFold(parsed.Status, "authorized") ||
		strings.EqualFold(parsed.Status, "authorised") {

		log.Printf("[poller] consent %s approved ✅", c.ConsentID)
		repo.UpdateConsentStatus(c.ConsentID, "approved")
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

// 🔔 Уведомление всех, кто ждал конкретное согласие
func (p *Poller) notifySubscribers(consentID string) {
	p.mu.Lock()
	if ch, ok := p.subscribers[consentID]; ok {
		ch <- "approved"
		close(ch)
		delete(p.subscribers, consentID)
	}
	p.mu.Unlock()
}
