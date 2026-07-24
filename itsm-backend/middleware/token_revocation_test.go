package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddlewareRejectsRevokedAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := currentAccessTokenRevocationStore()
	setAccessTokenRevocationStore(newMemoryAccessTokenRevocationStore())
	t.Cleanup(func() { setAccessTokenRevocationStore(original) })

	const secret = "token-revocation-test-secret"
	token, err := GenerateAccessToken(7, "operator", "admin", 3, secret, time.Hour)
	require.NoError(t, err)

	router := gin.New()
	router.GET("/protected", AuthMiddleware(secret), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	firstRequest.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(first, firstRequest)
	require.Equal(t, http.StatusNoContent, first.Code)

	claims, err := ValidateAccessToken(token, secret)
	require.NoError(t, err)
	require.NotNil(t, claims.ExpiresAt)
	require.NoError(t, RevokeAccessToken(context.Background(), token, claims.ExpiresAt.Time))

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	secondRequest.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(second, secondRequest)
	require.NotEqual(t, http.StatusNoContent, second.Code)
	require.Contains(t, second.Body.String(), "token已失效")
}

func TestAccessTokenRevocationKeyDoesNotExposeToken(t *testing.T) {
	const token = "header.payload.signature"
	key := accessTokenRevocationKey(token)
	require.NotContains(t, key, token)
	require.Contains(t, key, accessTokenRevocationPrefix)
}
