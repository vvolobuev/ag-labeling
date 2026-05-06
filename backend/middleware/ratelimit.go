package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type slidingWindow struct {
	mu    sync.Mutex
	times []time.Time
}

func (s *slidingWindow) allow(max int, win time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-win)
	i := 0
	for _, t := range s.times {
		if t.After(cutoff) {
			break
		}
		i++
	}
	if i > 0 {
		s.times = s.times[i:]
	}
	if len(s.times) >= max {
		return false
	}
	s.times = append(s.times, now)
	return true
}

type syncMapLimiter struct {
	mu sync.Mutex
	m  map[string]*slidingWindow
}

func (l *syncMapLimiter) get(key string) *slidingWindow {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.m == nil {
		l.m = make(map[string]*slidingWindow)
	}
	w, ok := l.m[key]
	if !ok {
		w = &slidingWindow{}
		l.m[key] = w
	}
	return w
}

func SlidingWindowRateLimit(max int, win time.Duration, keyFn func(*gin.Context) string) gin.HandlerFunc {
	var lim syncMapLimiter
	return func(c *gin.Context) {
		k := keyFn(c)
		if k == "" {
			k = "unknown"
		}
		if !lim.get(k).allow(max, win) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, try again later"})
			return
		}
		c.Next()
	}
}

func ClientIPKey(c *gin.Context) string {
	return c.ClientIP()
}
