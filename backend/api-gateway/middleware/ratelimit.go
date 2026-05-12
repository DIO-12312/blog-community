package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	mu             sync.Mutex
	capacity       int       // 桶容量
	tokens         float64   // 当前令牌数
	refillRate     float64   // 每秒补充令牌数
	lastRefillTime time.Time // 上次补充时间
}

// NewTokenBucket 创建令牌桶
// capacity: 桶容量（最多同时处理多少请求）
// refillRate: 每秒补充的令牌数（QPS）
func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:       capacity,
		tokens:         float64(capacity),
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

// TryAcquire 尝试获取一个令牌（非阻塞）
func (tb *TokenBucket) TryAcquire() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 计算应该补充的令牌数
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()
	newTokens := elapsed * tb.refillRate

	// 补充令牌，但不超过容量
	tb.tokens = min(float64(tb.capacity), tb.tokens+newTokens)
	tb.lastRefillTime = now

	// 如果有令牌，消耗一个
	if tb.tokens >= 1.0 {
		tb.tokens--
		return true
	}

	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimitMiddleware 返回限流中间件
// 基于 IP 的限流：不同 IP 有不同的限额
func RateLimitMiddleware() gin.HandlerFunc {
	limiters := make(map[string]*TokenBucket)
	mu := sync.RWMutex{}

	// 每个 IP 每秒最多 10 个请求，桶容量 20（允许短时突发）
	const qps = 10.0
	const capacity = 20

	return func(c *gin.Context) {
		// 获取客户端 IP
		clientIP := getClientIP(c)

		mu.Lock()
		limiter, exists := limiters[clientIP]
		if !exists {
			limiter = NewTokenBucket(capacity, qps)
			limiters[clientIP] = limiter
		}
		mu.Unlock()

		// 尝试获取令牌
		if !limiter.TryAcquire() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}

		c.Next()
	}
}

// getClientIP 获取客户端 IP（考虑代理）
func getClientIP(c *gin.Context) string {
	// 优先从代理头获取真实 IP
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	// 直接连接的 IP
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ip
}
