package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	authenticationv1 "admin/api/gen/authentication/v1"
	cachepkg "github.com/chnxq/xkitpkg/cache"
	conf "github.com/chnxq/xkitpkg/conf/v1"
)

const (
	authFlowStoreKeyPrefix = "authflow"
	authFlowSessionTTL     = 10 * time.Minute
	authBindTokenTTL       = 10 * time.Minute
)

type authFlowSessionRecord struct {
	SessionID         string                               `json:"session_id"`
	SessionToken      string                               `json:"session_token"`
	Scene             authenticationv1.AuthFlowScene       `json:"scene"`
	Status            authenticationv1.AuthFlowStatus      `json:"status"`
	TenantID          uint32                               `json:"tenant_id,omitempty"`
	UserID            uint32                               `json:"user_id,omitempty"`
	ProviderKey       string                               `json:"provider_key,omitempty"`
	Provider          authenticationv1.OAuthProvider       `json:"provider,omitempty"`
	ProviderAccountID string                               `json:"provider_account_id,omitempty"`
	ClientType        authenticationv1.ClientType          `json:"client_type,omitempty"`
	RedirectURI       string                               `json:"redirect_uri,omitempty"`
	ExtraInfo         string                               `json:"extra_info,omitempty"`
	ExpiresAt         time.Time                            `json:"expires_at,omitempty"`
	ConfirmedAt       *time.Time                           `json:"confirmed_at,omitempty"`
	QRCodeURL         string                               `json:"qr_code_url,omitempty"`
	DisplayHint       string                               `json:"display_hint,omitempty"`
	BindToken         string                               `json:"bind_token,omitempty"`
	Profile           *authFlowProviderProfile             `json:"profile,omitempty"`
}

type authFlowProviderProfile struct {
	ProviderKey       string                         `json:"provider_key,omitempty"`
	Provider          authenticationv1.OAuthProvider `json:"provider,omitempty"`
	ProviderAccountID string                         `json:"provider_account_id,omitempty"`
	Nickname          string                         `json:"nickname,omitempty"`
	Avatar            string                         `json:"avatar,omitempty"`
	Email             string                         `json:"email,omitempty"`
	Mobile            string                         `json:"mobile,omitempty"`
	RawProfileJSON    string                         `json:"raw_profile_json,omitempty"`
}

type authFlowBindRecord struct {
	SessionID   string                   `json:"session_id"`
	TenantID    uint32                   `json:"tenant_id,omitempty"`
	ProviderKey string                   `json:"provider_key,omitempty"`
	Provider    authenticationv1.OAuthProvider `json:"provider,omitempty"`
	Profile     *authFlowProviderProfile `json:"profile,omitempty"`
	ExpiresAt   time.Time                `json:"expires_at,omitempty"`
}

type authFlowStore struct {
	cache cachepkg.AdapterCache
}

var (
	sharedAuthFlowStoreMu   sync.Mutex
	sharedAuthFlowStore     *authFlowStore
	sharedAuthFlowStoreErr  error
	sharedAuthFlowStoreInit bool
)

func newAuthFlowStore(dataCfg *conf.Data) (*authFlowStore, error) {
	sharedAuthFlowStoreMu.Lock()
	defer sharedAuthFlowStoreMu.Unlock()

	if sharedAuthFlowStoreInit {
		return sharedAuthFlowStore, sharedAuthFlowStoreErr
	}
	sharedAuthFlowStore, sharedAuthFlowStoreErr = newStandaloneAuthFlowStore(dataCfg)
	sharedAuthFlowStoreInit = true
	return sharedAuthFlowStore, sharedAuthFlowStoreErr
}

func newStandaloneAuthFlowStore(dataCfg *conf.Data) (*authFlowStore, error) {
	var (
		cache cachepkg.AdapterCache
		err   error
	)
	if dataCfg != nil {
		cache, err = cachepkg.NewCache(dataCfg)
		if err != nil {
			return nil, err
		}
	} else {
		cache = cachepkg.NewMemory()
		if err := cache.Connect(); err != nil {
			return nil, err
		}
	}
	return &authFlowStore{cache: cache}, nil
}

func (s *authFlowStore) SaveSession(record *authFlowSessionRecord) error {
	if s == nil || s.cache == nil {
		return fmt.Errorf("auth flow store is unavailable")
	}
	if record == nil || strings.TrimSpace(record.SessionID) == "" {
		return fmt.Errorf("auth flow session is required")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	ttl := time.Until(record.ExpiresAt)
	if ttl <= 0 {
		ttl = authFlowSessionTTL
	}
	return s.cache.Set(authFlowSessionKey(record.SessionID), string(payload), ttl)
}

func (s *authFlowStore) GetSession(sessionID string) (*authFlowSessionRecord, error) {
	if s == nil || s.cache == nil {
		return nil, fmt.Errorf("auth flow store is unavailable")
	}
	payload, err := s.cache.Get(authFlowSessionKey(sessionID))
	if err != nil || payload == "" {
		return nil, fmt.Errorf("auth flow session not found")
	}
	record := &authFlowSessionRecord{}
	if err := json.Unmarshal([]byte(payload), record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *authFlowStore) DeleteSession(sessionID string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.Del(authFlowSessionKey(sessionID))
}

func (s *authFlowStore) SaveBindRecord(bindToken string, record *authFlowBindRecord) error {
	if s == nil || s.cache == nil {
		return fmt.Errorf("auth flow store is unavailable")
	}
	if strings.TrimSpace(bindToken) == "" || record == nil {
		return fmt.Errorf("bind record is required")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	ttl := time.Until(record.ExpiresAt)
	if ttl <= 0 {
		ttl = authBindTokenTTL
	}
	return s.cache.Set(authFlowBindKey(bindToken), string(payload), ttl)
}

func (s *authFlowStore) GetBindRecord(bindToken string) (*authFlowBindRecord, error) {
	if s == nil || s.cache == nil {
		return nil, fmt.Errorf("auth flow store is unavailable")
	}
	payload, err := s.cache.Get(authFlowBindKey(bindToken))
	if err != nil || payload == "" {
		return nil, fmt.Errorf("bind record not found")
	}
	record := &authFlowBindRecord{}
	if err := json.Unmarshal([]byte(payload), record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *authFlowStore) DeleteBindRecord(bindToken string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.Del(authFlowBindKey(bindToken))
}

func authFlowSessionKey(sessionID string) string {
	return fmt.Sprintf("%s:session:%s", authFlowStoreKeyPrefix, strings.TrimSpace(sessionID))
}

func authFlowBindKey(bindToken string) string {
	return fmt.Sprintf("%s:bind:%s", authFlowStoreKeyPrefix, strings.TrimSpace(bindToken))
}
