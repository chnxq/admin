package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	authenticationv1 "admin/api/gen/authentication/v1"
	identityv1 "admin/api/gen/identity/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/user"
	"admin/internal/data/repo"
	"github.com/chnxq/xkitpkg/app"
	transport "github.com/chnxq/xkitpkg/transport"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenTTL  = 2 * time.Hour
	refreshTokenTTL = 7 * 24 * time.Hour
	accessTokenRefreshGraceTTL = 30 * time.Second

	tokenCategoryAccess  = "access"
	tokenIssuer          = "admin"
)

var publicHTTPPaths = map[string]struct{}{
	"/admin/v1/captcha":                                 {},
	"/admin/v1/login":                                   {},
	"/admin/v1/logout":                                  {},
	"/admin/v1/register":                                {},
	"/admin/v1/refresh-token":                           {},
	"/admin/v1/auth-sessions":                           {},
	"/admin/v1/auth-sessions/{session_id}":              {},
	"/admin/v1/auth-sessions/{session_id}/poll":         {},
	"/admin/v1/social-auth:start":                       {},
	"/admin/v1/social-auth:complete":                    {},
	"/admin/v1/social-auth/miniapp:exchange-code":       {},
	"/admin/v1/social-auth:confirm-bind-or-register":    {},
	"/docs":                                             {},
}

type authConfig struct {
	signingMethod jwtv5.SigningMethod
	secretKey     []byte
}

type userCredentialFinder interface {
	FindPasswordCredentialByIdentifier(context.Context, string) (*repo.UserCredentialWithUser, error)
	UpgradePasswordCredential(context.Context, uint32, string) error
}

type authTokenClaims struct {
	UserID    uint32   `json:"uid"`
	TenantID  uint32   `json:"tid,omitempty"`
	OrgUnitID uint32   `json:"ouid,omitempty"`
	Username  string   `json:"sub"`
	Roles     []string `json:"roles,omitempty"`
	Category  string   `json:"cat"`
	ClientID  string   `json:"cid,omitempty"`
	DeviceID  string   `json:"did,omitempty"`
	jwtv5.RegisteredClaims
}

func loadAuthConfig(appCtx *app.AppCtx) (*authConfig, error) {
	if appCtx == nil {
		return nil, fmt.Errorf("app context is required")
	}
	cfg := appCtx.GetConfig()
	if cfg == nil || cfg.GetAuthn() == nil {
		return nil, fmt.Errorf("auth config is missing")
	}
	authn := cfg.GetAuthn()
	if !strings.EqualFold(strings.TrimSpace(authn.GetType()), "jwt") {
		return nil, fmt.Errorf("unsupported authn type %q", authn.GetType())
	}
	jwtCfg := authn.GetJwt()
	if jwtCfg == nil {
		return nil, fmt.Errorf("jwt auth config is missing")
	}
	method, err := jwtSigningMethod(jwtCfg.GetMethod())
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(jwtCfg.GetKey())
	if key == "" {
		return nil, fmt.Errorf("jwt key is empty")
	}
	return &authConfig{
		signingMethod: method,
		secretKey:     []byte(key),
	}, nil
}

func jwtSigningMethod(name string) (jwtv5.SigningMethod, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "", "HS256":
		return jwtv5.SigningMethodHS256, nil
	case "HS384":
		return jwtv5.SigningMethodHS384, nil
	case "HS512":
		return jwtv5.SigningMethodHS512, nil
	default:
		return nil, fmt.Errorf("unsupported jwt signing method %q", name)
	}
}

func parseBearerToken(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok || tr == nil {
		return ""
	}
	authHeader := strings.TrimSpace(tr.RequestHeader().Get("Authorization"))
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func parseAndValidateToken(tokenString string, auth *authConfig, wantCategory string) (*authTokenClaims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, authenticationv1.ErrorUnauthorized("token is required")
	}
	if auth == nil {
		return nil, authenticationv1.ErrorInternalServerError("auth config is unavailable")
	}

	claims := &authTokenClaims{}
	token, err := jwtv5.ParseWithClaims(tokenString, claims, func(token *jwtv5.Token) (any, error) {
		if token.Method.Alg() != auth.signingMethod.Alg() {
			return nil, authenticationv1.ErrorUnauthorized("unexpected signing method")
		}
		return auth.secretKey, nil
	}, jwtv5.WithValidMethods([]string{auth.signingMethod.Alg()}))
	if err != nil {
		if errorsIsAny(err, jwtv5.ErrTokenExpired) {
			return nil, authenticationv1.ErrorTokenExpired("token expired")
		}
		return nil, authenticationv1.ErrorUnauthorized("invalid token")
	}
	if token == nil || !token.Valid {
		return nil, authenticationv1.ErrorUnauthorized("invalid token")
	}
	if wantCategory != "" && claims.Category != wantCategory {
		if wantCategory == tokenCategoryAccess {
			return nil, authenticationv1.ErrorIncorrectAccessToken("unexpected token category")
		}
		return nil, authenticationv1.ErrorUnauthorized("unexpected token category")
	}
	if strings.TrimSpace(claims.Username) == "" || claims.UserID == 0 {
		return nil, authenticationv1.ErrorUnauthorized("token payload is incomplete")
	}
	return claims, nil
}

func buildTokenClaims(userDTO *identityv1.User, category string, req *authenticationv1.LoginRequest, now time.Time, ttl time.Duration) *authTokenClaims {
	jti := strings.TrimSpace(req.GetJti())
	if jti == "" {
		jti = newJWTID()
	}
	claims := &authTokenClaims{
		UserID:    userDTO.GetId(),
		TenantID:  userDTO.GetTenantId(),
		OrgUnitID: userDTO.GetOrgUnitId(),
		Username:  userDTO.GetUsername(),
		Roles:     append([]string(nil), userDTO.GetRoles()...),
		Category:  category,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   userDTO.GetUsername(),
			ID:        jti,
			IssuedAt:  jwtv5.NewNumericDate(now),
			NotBefore: jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(ttl)),
		},
	}
	if req != nil {
		if clientID := strings.TrimSpace(req.GetClientId()); clientID != "" {
			claims.ClientID = clientID
		}
		if deviceID := strings.TrimSpace(req.GetDeviceId()); deviceID != "" {
			claims.DeviceID = deviceID
		}
	}
	return claims
}

func signToken(claims *authTokenClaims, auth *authConfig) (string, error) {
	if claims == nil {
		return "", fmt.Errorf("claims are required")
	}
	token := jwtv5.NewWithClaims(auth.signingMethod, claims)
	return token.SignedString(auth.secretKey)
}

func newJWTID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func buildAccessToken(userDTO *identityv1.User, req *authenticationv1.LoginRequest, auth *authConfig) (string, *authTokenClaims, error) {
	now := time.Now()
	accessClaims := buildTokenClaims(userDTO, tokenCategoryAccess, req, now, accessTokenTTL)
	accessToken, err := signToken(accessClaims, auth)
	if err != nil {
		return "", nil, err
	}
	return accessToken, accessClaims, nil
}

func buildLoginResponse(accessToken string, refreshToken string) (*authenticationv1.LoginResponse, error) {
	refreshExpiresIn := int64(refreshTokenTTL / time.Second)
	return &authenticationv1.LoginResponse{
		TokenType:        authenticationv1.TokenType_bearer,
		AccessToken:      accessToken,
		ExpiresIn:        int64(accessTokenTTL / time.Second),
		RefreshToken:     &refreshToken,
		RefreshExpiresIn: &refreshExpiresIn,
	}, nil
}

func buildRefreshTokenRecord(userDTO *identityv1.User, req *authenticationv1.LoginRequest, accessClaims *authTokenClaims) *refreshTokenRecord {
	if userDTO == nil || accessClaims == nil {
		return nil
	}
	record := &refreshTokenRecord{
		UserID:    userDTO.GetId(),
		TenantID:  userDTO.GetTenantId(),
		OrgUnitID: userDTO.GetOrgUnitId(),
		Username:  userDTO.GetUsername(),
		ClientID:  normalizedClientType(accessClaims.ClientID),
		DeviceID:  accessClaims.DeviceID,
		JTI:       accessClaims.ID,
	}
	if req != nil {
		if clientID := strings.TrimSpace(req.GetClientId()); clientID != "" {
			record.ClientID = clientID
		}
		if deviceID := strings.TrimSpace(req.GetDeviceId()); deviceID != "" {
			record.DeviceID = deviceID
		}
	}
	return record
}

func issueLoginResponse(userDTO *identityv1.User, req *authenticationv1.LoginRequest, auth *authConfig, store *tokenStore) (*authenticationv1.LoginResponse, error) {
	if userDTO == nil {
		return nil, authenticationv1.ErrorUserNotFound("user not found")
	}
	if auth == nil || store == nil {
		return nil, authenticationv1.ErrorInternalServerError("authentication service is unavailable")
	}
	accessToken, accessClaims, err := buildAccessToken(userDTO, req, auth)
	if err != nil {
		return nil, err
	}
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshRecord := buildRefreshTokenRecord(userDTO, req, accessClaims)
	if err := store.StoreTokenPair(accessToken, accessClaims, refreshToken, refreshRecord); err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to persist token pair")
	}
	return buildLoginResponse(accessToken, refreshToken)
}

func generateRefreshToken() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return "rt_" + hex.EncodeToString(randomBytes), nil
}

func isUserAllowedToLogin(userEntity *ent.User) error {
	if userEntity == nil || userEntity.Status == nil {
		return nil
	}
	switch *userEntity.Status {
	case user.StatusDisabled, user.StatusClosed:
		return authenticationv1.ErrorForbidden("user is disabled")
	case user.StatusLocked:
		return authenticationv1.ErrorUserFreeze("user is locked")
	case user.StatusExpired:
		return authenticationv1.ErrorForbidden("user is expired")
	default:
		return nil
	}
}

func isProtectedServerRequest(ctx context.Context) bool {
	tr, ok := transport.FromServerContext(ctx)
	if !ok || tr == nil {
		return false
	}
	if tr.Kind() == transport.KindGRPC {
		return true
	}
	httpTr, ok := tr.(httptransport.Transporter)
	if !ok || httpTr.Request() == nil {
		return false
	}
	pathTemplate := strings.TrimSpace(httpTr.PathTemplate())
	path := pathTemplate
	if path == "" {
		path = strings.TrimSpace(httpTr.Request().URL.Path)
	}
	if path == "" {
		return false
	}
	if path == "/docs" || strings.HasPrefix(path, "/docs/") || strings.HasPrefix(path, "/debug/pprof") {
		return false
	}
	if _, ok := publicHTTPPaths[path]; ok {
		return false
	}
	return strings.HasPrefix(path, "/admin/v1")
}

func errorsIsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if target != nil && errors.Is(err, target) {
			return true
		}
		if target != nil && strings.Contains(err.Error(), target.Error()) {
			return true
		}
	}
	return false
}
