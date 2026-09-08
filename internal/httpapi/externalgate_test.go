package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tristenlammi/arrmada/internal/auth"
)

// The Discover-only scope is for requesters, read-only accounts and strangers. A
// signed-in admin or manager gets the whole app from wherever they are — the owner
// opening the app through their own tunnel hostname was getting the requester's view.
func TestExternalGateExemptsStaff(t *testing.T) {
	a := &api{}
	var seen bool
	gate := a.externalGate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = isExternalRequest(r)
		w.WriteHeader(http.StatusOK)
	}))
	call := func(user *auth.User) (int, bool) {
		r := httptest.NewRequest("GET", "http://x/api/v1/series", nil)
		r.RemoteAddr = "203.0.113.50:5000" // internet visitor
		if user != nil {
			r = withUser(r, user)
		}
		rec := httptest.NewRecorder()
		seen = false
		gate.ServeHTTP(rec, r)
		return rec.Code, seen
	}
	if code, _ := call(nil); code != http.StatusForbidden {
		t.Errorf("anonymous internet visitor reached a LAN-only endpoint (HTTP %d)", code)
	}
	if code, _ := call(&auth.User{Role: auth.RoleRequester}); code != http.StatusForbidden {
		t.Errorf("requester from the internet reached a LAN-only endpoint (HTTP %d)", code)
	}
	if code, ext := call(&auth.User{Role: auth.RoleAdmin}); code != http.StatusOK || ext {
		t.Errorf("admin from the internet: HTTP %d, external=%v; want the full app", code, ext)
	}
	if code, ext := call(&auth.User{Role: auth.RoleManager}); code != http.StatusOK || ext {
		t.Errorf("manager from the internet: HTTP %d, external=%v; want the full app", code, ext)
	}
	if code, _ := call(&auth.User{Role: auth.RoleAdmin, Disabled: true}); code != http.StatusForbidden {
		t.Errorf("a disabled admin account still bypassed the gate (HTTP %d)", code)
	}
	// Discover stays reachable for everyone outside.
	r := httptest.NewRequest("GET", "http://x/api/v1/discover/trending", nil)
	r.RemoteAddr = "203.0.113.50:5000"
	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Errorf("Discover blocked for an internet visitor (HTTP %d)", rec.Code)
	}
}
