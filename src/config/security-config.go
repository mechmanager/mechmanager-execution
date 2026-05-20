package security

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func NewGinEngine(nrApp *newrelic.Application) *gin.Engine {
	r := gin.Default()

	if nrApp != nil {
		r.Use(nrgin.Middleware(nrApp))
	}

	origins := os.Getenv("CORS_ALLOW_ORIGINS")
	allowOrigins := []string{"http://localhost"}
	if origins != "" {
		allowOrigins = strings.Split(origins, ",")
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	return r
}
