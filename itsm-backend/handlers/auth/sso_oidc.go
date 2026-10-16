package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OIDC 客户端：Discovery、JWKS、授权码交换、id_token 验签。
//
// 安全口径：
//   - 只接受非对称签名（RS/ES/PS/EdDSA）。"none" 与 HS* 一律拒绝：
//     HS256 用 client_secret 当 HMAC 密钥，任何知道公开 client_id 语义的一方
//     都能自签 token，等于把 IdP 降级成「谁都可以冒充」。
//   - 验签同时校验 iss / aud / exp，nonce 与 state 绑定的值比对（缺失只告警不阻断，
//     见 verifyIDToken 注释）。
//   - JWKS 按 kid 惰性拉取并缓存；遇到未知 kid 强制刷新一次，防密钥轮换期 401。

const (
	oidcHTTPTimeout      = 10 * time.Second
	oidcCacheTTL         = time.Hour
	oidcDiscoveryTimeout = 10 * time.Second
)

// allowedAlgs 是 id_token 签名算法白名单（jwt.WithValidMethods 消费）。
var allowedOIDCAlgs = []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "PS256", "PS384", "PS512", "EdDSA"}

// oidcEndpoints 是完成一次授权码flow所需的全部端点。
type oidcEndpoints struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
}

// oidcIdentity 是从 id_token / userinfo 归一出的外部身份。
type oidcIdentity struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	Nonce             string `json:"nonce"`
}

type oidcClient struct {
	httpClient *http.Client

	mu        sync.Mutex
	endpoints map[int]cachedEndpoints // providerID -> 端点缓存
	jwks      map[string]cachedJWKS   // jwksURI -> 密钥缓存
}

type cachedEndpoints struct {
	endpoints oidcEndpoints
	fetchedAt time.Time
}

type cachedJWKS struct {
	keys      map[string]interface{} // kid -> 公钥
	fetchedAt time.Time
}

func newOIDCClient() *oidcClient {
	return &oidcClient{
		httpClient: &http.Client{Timeout: oidcHTTPTimeout},
		endpoints:  make(map[int]cachedEndpoints),
		jwks:       make(map[string]cachedJWKS),
	}
}

// discoveryDocument 只取需要的字段（.well-known/openid-configuration 的超集忽略）。
type discoveryDocument struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// resolveEndpoints 返回提供方的有效端点：显式配置优先，缺失项由 Discovery 补齐。
// 两者都拿不到 token/jwks 端点时返回错误——没有 JWKS 就无法验签，
// 此时必须失败，绝不退化成「不验签也继续」。
func (c *oidcClient) resolveEndpoints(ctx context.Context, providerID int, issuer, authzURL, tokenURL, userinfoURL, jwksURL string) (oidcEndpoints, error) {
	endpoints := oidcEndpoints{
		AuthorizationEndpoint: strings.TrimSpace(authzURL),
		TokenEndpoint:         strings.TrimSpace(tokenURL),
		UserinfoEndpoint:      strings.TrimSpace(userinfoURL),
		JWKSURI:               strings.TrimSpace(jwksURL),
		Issuer:                strings.TrimSpace(issuer),
	}
	if endpoints.AuthorizationEndpoint != "" && endpoints.TokenEndpoint != "" && endpoints.JWKSURI != "" {
		return endpoints, nil
	}

	cached, ok := c.cachedEndpointsFor(providerID)
	if ok {
		endpoints = mergeEndpoints(endpoints, cached)
		if endpoints.AuthorizationEndpoint != "" && endpoints.TokenEndpoint != "" && endpoints.JWKSURI != "" {
			return endpoints, nil
		}
	}

	doc, err := c.fetchDiscovery(ctx, endpoints.Issuer)
	if err != nil {
		if ok {
			// 旧缓存能补齐剩余缺口时仍放行（Discovery 临时故障不该打死登录）。
			endpoints = mergeEndpoints(endpoints, cached)
			if endpoints.AuthorizationEndpoint != "" && endpoints.TokenEndpoint != "" && endpoints.JWKSURI != "" {
				return endpoints, nil
			}
		}
		return oidcEndpoints{}, fmt.Errorf("OIDC discovery 失败且端点配置不完整: %w", err)
	}
	// Discovery 的 issuer 必须与配置一致，否则可能是中间人替换的发现文档。
	if doc.Issuer != "" && doc.Issuer != endpoints.Issuer {
		return oidcEndpoints{}, fmt.Errorf("OIDC discovery issuer 不匹配: 期望 %q，实际 %q", endpoints.Issuer, doc.Issuer)
	}
	endpoints = mergeEndpoints(endpoints, oidcEndpoints{
		AuthorizationEndpoint: doc.AuthorizationEndpoint,
		TokenEndpoint:         doc.TokenEndpoint,
		UserinfoEndpoint:      doc.UserinfoEndpoint,
		JWKSURI:               doc.JWKSURI,
		Issuer:                endpoints.Issuer,
	})

	if endpoints.TokenEndpoint == "" || endpoints.JWKSURI == "" || endpoints.AuthorizationEndpoint == "" {
		return oidcEndpoints{}, fmt.Errorf("OIDC 端点不完整：authorization=%t token=%t jwks=%t（请补全提供方配置或修复 Discovery 文档）",
			endpoints.AuthorizationEndpoint != "", endpoints.TokenEndpoint != "", endpoints.JWKSURI != "")
	}

	c.mu.Lock()
	c.endpoints[providerID] = cachedEndpoints{endpoints: endpoints, fetchedAt: time.Now()}
	c.mu.Unlock()
	return endpoints, nil
}

func mergeEndpoints(explicit, fallback oidcEndpoints) oidcEndpoints {
	if explicit.AuthorizationEndpoint == "" {
		explicit.AuthorizationEndpoint = fallback.AuthorizationEndpoint
	}
	if explicit.TokenEndpoint == "" {
		explicit.TokenEndpoint = fallback.TokenEndpoint
	}
	if explicit.UserinfoEndpoint == "" {
		explicit.UserinfoEndpoint = fallback.UserinfoEndpoint
	}
	if explicit.JWKSURI == "" {
		explicit.JWKSURI = fallback.JWKSURI
	}
	return explicit
}

func (c *oidcClient) cachedEndpointsFor(providerID int) (oidcEndpoints, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.endpoints[providerID]
	if !ok || time.Since(entry.fetchedAt) > oidcCacheTTL {
		return oidcEndpoints{}, false
	}
	return entry.endpoints, true
}

func (c *oidcClient) fetchDiscovery(ctx context.Context, issuer string) (discoveryDocument, error) {
	if issuer == "" {
		return discoveryDocument{}, fmt.Errorf("issuer 为空，无法执行 OIDC discovery")
	}
	discoveryURL := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	reqCtx, cancel := context.WithTimeout(ctx, oidcDiscoveryTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return discoveryDocument{}, fmt.Errorf("构造 discovery 请求失败: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return discoveryDocument{}, fmt.Errorf("请求 %s 失败: %w", discoveryURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return discoveryDocument{}, fmt.Errorf("discovery 端点返回 %d", resp.StatusCode)
	}
	var doc discoveryDocument
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil {
		return discoveryDocument{}, fmt.Errorf("解析 discovery 文档失败: %w", err)
	}
	return doc, nil
}

// jwksDocument 只解析验签所需字段。
type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Crv string `json:"crv"`
	N   string `json:"n"`
	E   string `json:"e"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

// jwksKeys 返回 JWKS 密钥集合；forceRefresh 用于未知 kid 的密钥轮换场景。
func (c *oidcClient) jwksKeys(ctx context.Context, jwksURI string, forceRefresh bool) (map[string]interface{}, error) {
	c.mu.Lock()
	entry, ok := c.jwks[jwksURI]
	c.mu.Unlock()
	if ok && !forceRefresh && time.Since(entry.fetchedAt) <= oidcCacheTTL {
		return entry.keys, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return nil, fmt.Errorf("构造 JWKS 请求失败: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 JWKS 失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS 端点返回 %d", resp.StatusCode)
	}
	var doc jwksDocument
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("解析 JWKS 失败: %w", err)
	}
	keys := make(map[string]interface{}, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Use != "" && k.Use != "sig" {
			continue // 加密用途密钥不用于验签
		}
		pub, err := jwkToPublicKey(k)
		if err != nil {
			continue // 单条无法解析不阻断其余密钥
		}
		if k.Kid != "" {
			keys[k.Kid] = pub
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("JWKS 未包含任何可用签名密钥")
	}
	c.mu.Lock()
	c.jwks[jwksURI] = cachedJWKS{keys: keys, fetchedAt: time.Now()}
	c.mu.Unlock()
	return keys, nil
}

func jwkToPublicKey(k jwkKey) (interface{}, error) {
	switch k.Kty {
	case "RSA":
		if k.N == "" || k.E == "" {
			return nil, fmt.Errorf("RSA JWK 缺少 n/e")
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, err
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, err
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		if e == 0 {
			return nil, fmt.Errorf("RSA JWK 指数为 0")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
	case "EC":
		if k.Crv == "" || k.X == "" || k.Y == "" {
			return nil, fmt.Errorf("EC JWK 缺少 crv/x/y")
		}
		var ecCurve elliptic.Curve
		switch k.Crv {
		case "P-256":
			ecCurve = elliptic.P256()
		case "P-384":
			ecCurve = elliptic.P384()
		case "P-521":
			ecCurve = elliptic.P521()
		default:
			return nil, fmt.Errorf("不支持的 EC 曲线 %s", k.Crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, err
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{Curve: ecCurve, X: new(big.Int).SetBytes(xBytes), Y: new(big.Int).SetBytes(yBytes)}, nil
	case "OKP":
		if k.Crv != "Ed25519" || k.X == "" {
			return nil, fmt.Errorf("仅支持 Ed25519 OKP 密钥")
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, err
		}
		if len(xBytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("Ed25519 公钥长度非法")
		}
		return ed25519.PublicKey(xBytes), nil
	default:
		return nil, fmt.Errorf("不支持的 JWK 类型 %s", k.Kty)
	}
}

// tokenResponse 是授权码交换的响应（只取需要的字段）。
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// exchangeCode 用授权码换 token。PKCE verifier 必须原样携带，
// 否则启用了 PKCE 的 IdP 会直接拒绝（这也是防授权码拦截的机制）。
func (c *oidcClient) exchangeCode(ctx context.Context, endpoints oidcEndpoints, clientID, clientSecret, code, redirectURI, verifier string) (*tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", clientID)
	if verifier != "" {
		form.Set("code_verifier", verifier)
	}
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoints.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构造令牌请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("令牌交换请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取令牌响应失败: %w", err)
	}
	var tokenResp tokenResponse
	_ = json.Unmarshal(body, &tokenResp) // 非 JSON 错误体时靠状态码分支兜底
	if resp.StatusCode != http.StatusOK {
		if tokenResp.Error != "" {
			return nil, fmt.Errorf("IdP 拒绝授权码: %s: %s", tokenResp.Error, tokenResp.ErrorDesc)
		}
		return nil, fmt.Errorf("令牌交换返回 %d", resp.StatusCode)
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("IdP 拒绝授权码: %s: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if tokenResp.AccessToken == "" && tokenResp.IDToken == "" {
		return nil, fmt.Errorf("令牌响应既无 access_token 也无 id_token")
	}
	return &tokenResp, nil
}

// verifyIDToken 验签并解析 id_token。
// nonce 缺失时只告警不阻断：部分企业 IdP（旧版 ADFS 等）不回填 nonce，
// state 单次消费 + PKCE 已提供等价的重放防护；存在则必须与发起时一致。
func (c *oidcClient) verifyIDToken(ctx context.Context, endpoints oidcEndpoints, issuer, clientID, idToken, expectedNonce string, warn func(format string, args ...interface{})) (*oidcIdentity, error) {
	keys, err := c.jwksKeys(ctx, endpoints.JWKSURI, false)
	if err != nil {
		return nil, fmt.Errorf("获取 JWKS 失败: %w", err)
	}

	identity, err := parseAndVerifyIDToken(idToken, keys, issuer, clientID, expectedNonce)
	if err != nil {
		if isUnknownKid(err) {
			// 密钥轮换期：强制刷新 JWKS 重试一次。
			keys, kErr := c.jwksKeys(ctx, endpoints.JWKSURI, true)
			if kErr != nil {
				return nil, fmt.Errorf("JWKS 强制刷新失败: %w", kErr)
			}
			identity, err = parseAndVerifyIDToken(idToken, keys, issuer, clientID, expectedNonce)
		}
		if err != nil {
			return nil, err
		}
	}
	if identity.Nonce == "" && expectedNonce != "" && warn != nil {
		warn("id_token 未包含 nonce 声明，重放防护退化为 state+PKCE")
	}
	return identity, nil
}

func parseAndVerifyIDToken(idToken string, keys map[string]interface{}, issuer, clientID, expectedNonce string) (*oidcIdentity, error) {
	identity := &oidcIdentity{}
	token, err := jwt.Parse(idToken, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		key, ok := keys[kid]
		if !ok {
			// 无 kid 的 IdP：单密钥场景取唯一一把。
			if kid == "" && len(keys) == 1 {
				for _, only := range keys {
					return only, nil
				}
			}
			return nil, &unknownKidError{kid: kid}
		}
		return key, nil
	}, jwt.WithValidMethods(allowedOIDCAlgs), jwt.WithIssuer(issuer), jwt.WithAudience(clientID), jwt.WithLeeway(60*time.Second))
	if err != nil {
		return nil, fmt.Errorf("id_token 验签失败: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("id_token 无效")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("id_token claims 类型异常")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("id_token 缺少 sub 声明")
	}
	identity.Subject = sub
	identity.Email, _ = claims["email"].(string)
	if verified, ok := claims["email_verified"].(bool); ok {
		identity.EmailVerified = verified
	}
	identity.PreferredUsername, _ = claims["preferred_username"].(string)
	identity.Name, _ = claims["name"].(string)
	identity.Nonce, _ = claims["nonce"].(string)
	// nonce 存在但与发起时不符 → 拒绝（重放/混淆攻击）。
	// nonce 完全缺失 → 只告警不阻断：部分企业 IdP 不回填 nonce，
	// state 单次消费 + PKCE 已提供等价的重放防护（见 verifyIDToken 注释）。
	if expectedNonce != "" && identity.Nonce != "" && identity.Nonce != expectedNonce {
		return nil, fmt.Errorf("id_token nonce 与发起时不符")
	}
	return identity, nil
}

type unknownKidError struct {
	kid string
}

func (e *unknownKidError) Error() string {
	return fmt.Sprintf("JWKS 中不存在 kid=%q 的签名密钥", e.kid)
}

func isUnknownKid(err error) bool {
	var uke *unknownKidError
	return errors.As(err, &uke)
}

// fetchUserinfo 是 id_token 缺失时的兜底身份源。
func (c *oidcClient) fetchUserinfo(ctx context.Context, endpoints oidcEndpoints, accessToken string) (*oidcIdentity, error) {
	if endpoints.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("IdP 未返回 id_token 且未配置 userinfo 端点，无法获取身份")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoints.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("构造 userinfo 请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo 请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo 端点返回 %d", resp.StatusCode)
	}
	var identity oidcIdentity
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&identity); err != nil {
		return nil, fmt.Errorf("解析 userinfo 失败: %w", err)
	}
	if identity.Subject == "" {
		return nil, fmt.Errorf("userinfo 响应缺少 sub")
	}
	return &identity, nil
}
