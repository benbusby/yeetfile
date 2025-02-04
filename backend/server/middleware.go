package server

import (
	"fmt"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/time/rate"
	"net/http"
	"sync"
	"time"
	"yeetfile/backend/config"
	"yeetfile/backend/server/auth"
	"yeetfile/backend/server/session"
	"yeetfile/backend/utils"
	"yeetfile/shared/endpoints"
)

type Visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var visitors = make(map[[32]byte]*Visitor)
var mu sync.Mutex

const csp = "" +
	"default-src 'self';" +
	"img-src 'self' https://docs.yeetfile.com blob: data:;" +
	"media-src 'self' blob: data:;" +
	"script-src 'self' 'wasm-unsafe-eval';" +
	"style-src 'self' 'unsafe-inline';"

// getVisitor checks to see if an identifier (ip address or user id) is
// associated with a rate limiter, and returns it if so. If not, it creates a
// new entry in the visitors map associating the ip address with a new rate limiter.
func getVisitor(identifier string, path string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	idHash := blake2b.Sum256([]byte(identifier + path))
	visitor, exists := visitors[idHash]
	if !exists {
		limit := rate.Every(time.Second * time.Duration(config.YeetFileConfig.LimiterSeconds))
		limiter := rate.NewLimiter(limit, config.YeetFileConfig.LimiterAttempts)
		visitors[idHash] = &Visitor{limiter, time.Now()}
		return limiter
	}

	visitor.lastSeen = time.Now()
	return visitor.limiter
}

// LimiterMiddleware restricts requests to a particular route to prevent abuse
// of a handler function.
func LimiterMiddleware(next http.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, req *http.Request) {
		ip, err := utils.GetReqSource(req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		limiter := getVisitor(ip, req.URL.Path)
		if limiter.Allow() {
			next.ServeHTTP(w, req)
			return
		}

		http.Error(
			w,
			"Too many requests from this IP address -- please wait and try again",
			http.StatusTooManyRequests)
	}

	return handler
}

// AuthMiddleware enforces that a particular request has a valid session before
// handling.
func AuthMiddleware(next session.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if session.IsValidSession(w, req) {
			// Call the next handler
			id, err := session.GetSessionAndUserID(req)
			if err != nil {
				return
			}

			next(w, req, id)
			return
		}

		loginURL := fmt.Sprintf("%s?next=%s", endpoints.HTMLLogin, req.URL.Path)
		redirect := fmt.Sprintf(`
<html>
<head>
<meta http-equiv="refresh" content="0;URL='%[1]s'"/>
</head>
<body><p>Moved to <a href="%[1]s">%[1]s</a>.</p></body>
</html>`, loginURL)

		w.Write([]byte(redirect))
		return
	}

	return handler
}

// AdminMiddleware enforces that particular requests are only performed by those
// marked as "admin" in the database.
func AdminMiddleware(next session.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if session.IsValidSession(w, req) {
			id, err := session.GetSessionAndUserID(req)
			if err != nil {
				return
			}

			isAdmin := auth.IsInstanceAdmin(id)
			if isAdmin {
				// Call the next handler
				next(w, req, id)
				return
			}
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	return handler
}

// NoAuthMiddleware enforces that a particular request does NOT have a valid
// session before handling.
func NoAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if !session.HasSession(req) {
			next(w, req)
			return
		}

		redirectURL := string(endpoints.HTMLAccount)
		redirectCode := http.StatusTemporaryRedirect
		http.Redirect(w, req, redirectURL, redirectCode)
		return
	}

	return handler
}

// DefaultHeadersMiddleware applies headers to every route, regardless of
// other middlewares already applied.
func DefaultHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cspHeader := csp
		if utils.IsTLSReq(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
			w.Header().Set("Expect-CT", "max-age=86400, enforce")
			cspHeader += "frame-src 'self'"
		} else {
			// Required by StreamSaver.js in non-https contexts
			cspHeader += "frame-src 'self' https://jimmywarting.github.io/"
		}

		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", cspHeader)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		next.ServeHTTP(w, r)
	}
}

// AuthLimiterMiddleware is like AuthMiddleware, but also restricts requests to
// the same config.LimiterAttempts per config.LimiterSeconds by session
// (unlike LimiterMiddleware which limits by IP address)
func AuthLimiterMiddleware(next session.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, req *http.Request) {
		// Skip auth if the app is in debug mode, otherwise validate session
		if session.IsValidSession(w, req) {
			// Call the next handler
			id, err := session.GetSessionAndUserID(req)
			if err != nil {
				return
			}

			limiter := getVisitor(id, req.URL.Path)
			if limiter.Allow() {
				next(w, req, id)
				return
			} else {
				http.Error(
					w,
					"Too many requests from this account -- please wait and try again",
					http.StatusTooManyRequests)
				return
			}
		}

		http.Redirect(w, req, string(endpoints.HTMLLogin), http.StatusTemporaryRedirect)
		return
	}

	return handler
}

// StripeMiddleware ensures that requests made to Stripe related endpoints are
// only processed if Stripe has been set up already.
func StripeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if !config.YeetFileConfig.StripeBilling.Configured {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		next(w, req)
	}

	return handler
}

// BTCPayMiddleware ensures that requests made to BTCPay related endpoints are
// only processed if BTCPay has been set up already.
func BTCPayMiddleware(next http.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if !config.YeetFileConfig.BTCPayBilling.Configured {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		next(w, req)
	}

	return handler
}

// ManageLimiters removes an id->visitor pairing from the visitors map if they
// haven't repeated a limiter-enabled request in constants.LimiterSeconds.
func ManageLimiters() {
	mu.Lock()
	for ip, v := range visitors {
		if time.Since(v.lastSeen) > time.Minute {
			delete(visitors, ip)
		}
	}
	mu.Unlock()
}
