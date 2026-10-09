package alertspush

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SherClockHolmes/webpush-go"
)

// TestWebPushSenderVAPIDSubject pins the VAPID JWT the real sender emits.
// webpush-go prefixes "mailto:" to any subscriber that is not an https:
// URL, so passing "mailto:..." produced sub="mailto:mailto:..." — an invalid
// contact Apple's push service rejects (BadJwtToken). Also pins the
// aes128gcm encoding Apple requires.
func TestWebPushSenderVAPIDSubject(t *testing.T) {
	var authz, encoding string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz = r.Header.Get("Authorization")
		encoding = r.Header.Get("Content-Encoding")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}
	browser, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("browser key: %v", err)
	}
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		t.Fatalf("auth secret: %v", err)
	}
	s := NewWebPushSender(pub, priv)
	s.client = srv.Client()
	status, err := s.Send(context.Background(), Subscription{
		Endpoint: srv.URL + "/push/abc",
		P256dh:   base64.RawURLEncoding.EncodeToString(browser.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(secret),
	}, []byte(`{"id":"t"}`))
	if err != nil || status != http.StatusCreated {
		t.Fatalf("Send = %d, %v", status, err)
	}
	if encoding != "aes128gcm" {
		t.Errorf("Content-Encoding = %q, want aes128gcm (Apple rejects aesgcm)", encoding)
	}
	if !strings.HasPrefix(authz, "vapid t=") {
		t.Fatalf("Authorization = %q, want the vapid scheme", authz)
	}
	jwt := strings.TrimPrefix(strings.SplitN(authz, ",", 2)[0], "vapid t=")
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT has %d parts", len(parts))
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		t.Fatalf("claims json: %v", err)
	}
	if claims.Sub != "mailto:alerts@cosmolabs.org" {
		t.Errorf("JWT sub = %q, want mailto:alerts@cosmolabs.org", claims.Sub)
	}
}
