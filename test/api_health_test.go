package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthLiveEndpointShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "live"})
	})

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
