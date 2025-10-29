package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"time"
)

// Sign computes base64 HMAC-SHA256 over method + path + body + timestamp.
func Sign(secret []byte, method, path string, body []byte, timestamp string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(method))
	m.Write([]byte(path))
	m.Write(body)
	m.Write([]byte(timestamp))
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

// Verify compares provided signature with freshly computed one using constant-time compare.
func Verify(secret []byte, method, path string, body []byte, timestamp, providedSig string) bool {
	expected := Sign(secret, method, path, body, timestamp)
	exb, err1 := base64.StdEncoding.DecodeString(expected)
	pb, err2 := base64.StdEncoding.DecodeString(providedSig)
	if err1 != nil || err2 != nil {
		return false
	}
	return hmac.Equal(exb, pb)
}

// VerifyHMACMiddleware validates X-Timestamp and X-Signature for incoming requests.
// It reads the body once, restores it for handlers, and enforces max clock skew.
func VerifyHMACMiddleware(secret []byte, maxSkew time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ts := r.Header.Get("X-Timestamp")
			sig := r.Header.Get("X-Signature")
			if ts == "" || sig == "" {
				http.Error(w, "missing signature", http.StatusUnauthorized)
				return
			}
			pt, err := time.Parse(time.RFC3339, ts)
			if err != nil || time.Since(pt) > maxSkew || time.Until(pt) > maxSkew {
				http.Error(w, "invalid timestamp", http.StatusUnauthorized)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
			if !Verify(secret, r.Method, r.URL.Path, body, ts, sig) {
				http.Error(w, "bad signature", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
