package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	authenticationv1 "admin/api/gen/authentication/v1"
	cachepkg "github.com/chnxq/xkitpkg/cache"
	conf "github.com/chnxq/xkitpkg/conf/v1"
)

const (
	tokenStoreKeyPrefix = "auth"
	defaultClientType   = "admin"
)

type tokenStore struct {
	cache cachepkg.AdapterCache
}

var (
	sharedTokenStoreMu   sync.Mutex
	sharedTokenStore     *tokenStore
	sharedTokenStoreErr  error
	sharedTokenStoreInit bool
)

type refreshTokenRecord struct {
	UserID    uint32 `json:"user_id"`
	TenantID  uint32 `json:"tenant_id,omitempty"`
	OrgUnitID uint32 `json:"org_unit_id,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	Username  string `json:"username,omitempty"`
	JTI       string `json:"jti"`
}

type accessTokenRecord struct {
	UserID       uint32 `json:"user_id"`
	ClientID     string `json:"client_id,omitempty"`
	JTI          string `json:"jti"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func newTokenStore(dataCfg *conf.Data) (*tokenStore, error) {
	sharedTokenStoreMu.Lock()
	defer sharedTokenStoreMu.Unlock()

	if sharedTokenStoreInit {
		return sharedTokenStore, sharedTokenStoreErr
	}
	sharedTokenStore, sharedTokenStoreErr = newStandaloneTokenStore(dataCfg)
	sharedTokenStoreInit = true
	return sharedTokenStore, sharedTokenStoreErr
}

func newStandaloneTokenStore(dataCfg *conf.Data) (*tokenStore, error) {
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
	return &tokenStore{cache: cache}, nil
}

func (s *tokenStore) StoreTokenPair(accessToken string, accessClaims *authTokenClaims, refreshToken string, refreshRecord *refreshTokenRecord) error {
	if s == nil || s.cache == nil {
		return fmt.Errorf("token store is unavailable")
	}
	if accessClaims == nil || refreshRecord == nil {
		return fmt.Errorf("token claims are required")
	}
	if strings.TrimSpace(accessToken) == "" || strings.TrimSpace(refreshToken) == "" {
		return fmt.Errorf("token is required")
	}

	accessData, err := json.Marshal(accessTokenRecord{
		UserID:       accessClaims.UserID,
		ClientID:     normalizedClientType(accessClaims.ClientID),
		JTI:          accessClaims.ID,
		RefreshToken: refreshToken,
	})
	if err != nil {
		return err
	}
	refreshData, err := json.Marshal(refreshRecord)
	if err != nil {
		return err
	}

	if err := s.cache.Set(accessTokenKey(accessClaims.ID), accessToken, accessTokenTTL); err != nil {
		return err
	}
	if err := s.cache.Set(accessTokenMetaKey(accessClaims.ID), string(accessData), accessTokenTTL); err != nil {
		return err
	}
	if err := s.cache.Set(refreshTokenKey(refreshToken), string(refreshData), refreshTokenTTL); err != nil {
		return err
	}
	return nil
}

func (s *tokenStore) ValidateAccessToken(accessToken string, claims *authTokenClaims) error {
	if s == nil || s.cache == nil {
		return authenticationv1.ErrorUnauthorized("token store is unavailable")
	}
	if claims == nil || strings.TrimSpace(claims.ID) == "" {
		return authenticationv1.ErrorUnauthorized("invalid access token")
	}
	storedToken, err := s.cache.Get(accessTokenKey(claims.ID))
	if err != nil || storedToken == "" {
		return authenticationv1.ErrorUnauthorized("access token revoked")
	}
	if storedToken != accessToken {
		return authenticationv1.ErrorUnauthorized("access token revoked")
	}
	return nil
}

func (s *tokenStore) ResolveRefreshToken(refreshToken string) (*refreshTokenRecord, error) {
	if s == nil || s.cache == nil {
		return nil, authenticationv1.ErrorInternalServerError("token store is unavailable")
	}
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, authenticationv1.ErrorBadRequest("refresh_token is required")
	}
	payload, err := s.cache.Get(refreshTokenKey(refreshToken))
	if err != nil || payload == "" {
		return nil, authenticationv1.ErrorRefreshTokenNotFound("refresh token not found")
	}
	record := &refreshTokenRecord{}
	if err := json.Unmarshal([]byte(payload), record); err != nil {
		return nil, authenticationv1.ErrorIncorrectRefreshToken("invalid refresh token record")
	}
	if record.UserID == 0 || strings.TrimSpace(record.JTI) == "" {
		return nil, authenticationv1.ErrorIncorrectRefreshToken("refresh token payload is incomplete")
	}
	return record, nil
}

func (s *tokenStore) RevokeRefreshToken(refreshToken string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}
	return s.cache.Del(refreshTokenKey(refreshToken))
}

func (s *tokenStore) RevokeAccessTokenByJTI(jti string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	jti = strings.TrimSpace(jti)
	if jti == "" {
		return nil
	}
	if err := s.cache.Del(accessTokenKey(jti)); err != nil {
		return err
	}
	return s.cache.Del(accessTokenMetaKey(jti))
}

func (s *tokenStore) RevokeTokenPair(refreshToken string, jti string) error {
	if err := s.RevokeRefreshToken(refreshToken); err != nil {
		return err
	}
	return s.RevokeAccessTokenByJTI(jti)
}

func (s *tokenStore) RevokeTokenPairByJTI(jti string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	jti = strings.TrimSpace(jti)
	if jti == "" {
		return nil
	}
	metaValue, err := s.cache.Get(accessTokenMetaKey(jti))
	if err == nil && metaValue != "" {
		meta := &accessTokenRecord{}
		if json.Unmarshal([]byte(metaValue), meta) == nil && strings.TrimSpace(meta.RefreshToken) != "" {
			if err := s.RevokeRefreshToken(meta.RefreshToken); err != nil {
				return err
			}
		}
	}
	return s.RevokeAccessTokenByJTI(jti)
}

func normalizedClientType(clientID string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return defaultClientType
	}
	return clientID
}

func accessTokenKey(jti string) string {
	return fmt.Sprintf("%s:access:%s", tokenStoreKeyPrefix, strings.TrimSpace(jti))
}

func accessTokenMetaKey(jti string) string {
	return fmt.Sprintf("%s:access-meta:%s", tokenStoreKeyPrefix, strings.TrimSpace(jti))
}

func refreshTokenKey(refreshToken string) string {
	return fmt.Sprintf("%s:refresh:%s", tokenStoreKeyPrefix, strings.TrimSpace(refreshToken))
}
