package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/diploma/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ServiceProxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
	log    *zap.Logger
}

func New(targetURL string, log *zap.Logger) (*ServiceProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("proxy error",
			zap.String("target", targetURL),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"success":false,"error":"service temporarily unavailable"}`))
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("X-Powered-By")
		return nil
	}

	return &ServiceProxy{
		target: target,
		proxy:  proxy,
		log:    log,
	}, nil
}

func (sp *ServiceProxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.URL.Host = sp.target.Host
		c.Request.URL.Scheme = sp.target.Scheme
		c.Request.Header.Set("X-Forwarded-Host", c.Request.Header.Get("Host"))
		c.Request.Host = sp.target.Host
		sp.proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func HealthCheck(services map[string]string, log *zap.Logger) gin.HandlerFunc {
	client := &http.Client{Timeout: 3 * time.Second}

	return func(c *gin.Context) {
		statuses := make(map[string]interface{})
		allHealthy := true

		for name, serviceURL := range services {
			resp, err := client.Get(serviceURL + "/health")
			if err != nil {
				statuses[name] = gin.H{"status": "unhealthy", "error": err.Error()}
				allHealthy = false
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				statuses[name] = gin.H{"status": "healthy"}
			} else {
				statuses[name] = gin.H{"status": "unhealthy", "code": resp.StatusCode}
				allHealthy = false
			}
		}

		status := "healthy"
		httpStatus := http.StatusOK
		if !allHealthy {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, gin.H{
			"status":   status,
			"service":  "api-gateway",
			"services": statuses,
			"time":     time.Now().UTC(),
		})
	}
}

func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.NotFound(c, "route not found")
	}
}
