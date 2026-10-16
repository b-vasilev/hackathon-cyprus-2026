package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
)

func checkProductSharing(t *testing.T, body string, want bool) {
	t.Helper()
	var value access.Sharing
	if err := json.Unmarshal([]byte(body), &value); err != nil || value.SharedWithAll != want {
		t.Fatalf("sharing = %s, want %t: %v", body, want, err)
	}
}

func TestProductSharing(t *testing.T) {
	t.Parallel()
	portal := newTestPortal(t)
	request := portal.request
	path := "/api/products/" + portal.productID
	sharing := path + "/sharing"
	outsider := portal.register("existing@example.com")
	request(http.MethodPost, "/api/products", `{"name":"Private","sources":[{"kind":"text","text":"Private source"}]}`, portal.admin, http.StatusCreated)
	seedMCPPassport(t, portal, portal.productID, "demo", mcpFixture("Demo passport"))
	request(http.MethodPost, "/api/passport-revisions/demo/publish", "", portal.admin, http.StatusOK)

	checkMemberships := func(cookie *http.Cookie, role string) {
		t.Helper()
		var me access.Me
		if err := json.Unmarshal([]byte(request(http.MethodGet, "/api/auth/me", "", cookie, http.StatusOK)), &me); err != nil {
			t.Fatal(err)
		}
		if role == "" {
			if len(me.Memberships) != 0 {
				t.Fatalf("unexpected memberships: %+v", me.Memberships)
			}
			return
		}
		if len(me.Memberships) != 1 || me.Memberships[0].ProductID != portal.productID || me.Memberships[0].Role != role {
			t.Fatalf("memberships = %+v, want one %s role", me.Memberships, role)
		}
	}
	checkProducts := func(cookie *http.Cookie, wantVisible bool) {
		t.Helper()
		var products []product.Product
		if err := json.Unmarshal([]byte(request(http.MethodGet, "/api/products", "", cookie, http.StatusOK)), &products); err != nil {
			t.Fatal(err)
		}
		if wantVisible && len(products) == 1 && products[0].ID == portal.productID {
			return
		}
		if !wantVisible && len(products) == 0 {
			return
		}
		t.Fatalf("unexpected visible products: %+v", products)
	}

	// Products stay private by default. Only admins can enable broad sharing.
	checkProductSharing(t, request(http.MethodGet, sharing, "", portal.admin, http.StatusOK), false)
	request(http.MethodGet, sharing, "", nil, http.StatusUnauthorized)
	request(http.MethodPut, sharing, `{"shared_with_all":true}`, nil, http.StatusUnauthorized)
	for _, cookie := range []*http.Cookie{portal.lead, portal.learner, outsider} {
		request(http.MethodPut, sharing, `{"shared_with_all":true}`, cookie, http.StatusForbidden)
	}
	request(http.MethodPut, sharing, `{"shared_with_all":"true"}`, portal.admin, http.StatusBadRequest)
	request(http.MethodPut, "/api/products/missing/sharing", `{"shared_with_all":true}`, portal.admin, http.StatusNotFound)
	request(http.MethodGet, "/api/products/missing/sharing", "", portal.admin, http.StatusNotFound)
	request(http.MethodGet, path, "", outsider, http.StatusNotFound)
	checkProducts(outsider, false)
	checkMemberships(outsider, "")

	checkProductSharing(t, request(http.MethodPut, sharing, `{"shared_with_all":true}`, portal.admin, http.StatusOK), true)
	checkProductSharing(t, request(http.MethodGet, sharing, "", portal.lead, http.StatusOK), true)
	checkMemberships(portal.lead, access.RoleLead)
	checkMemberships(portal.learner, access.RoleLearner)

	// Existing sessions and newly registered accounts both receive learner access.
	newcomer := portal.register("future@example.com")
	for _, cookie := range []*http.Cookie{outsider, newcomer} {
		checkProducts(cookie, true)
		checkMemberships(cookie, access.RoleLearner)
		request(http.MethodGet, path, "", cookie, http.StatusOK)
		request(http.MethodGet, path+"/passport", "", cookie, http.StatusOK)
		request(http.MethodGet, path+"/passport.md", "", cookie, http.StatusOK)
		request(http.MethodGet, path+"/passport-revisions", "", cookie, http.StatusForbidden)
		request(http.MethodGet, "/api/passport-revisions/demo", "", cookie, http.StatusForbidden)
		request(http.MethodGet, path+"/members", "", cookie, http.StatusForbidden)
		request(http.MethodGet, sharing, "", cookie, http.StatusForbidden)
		request(http.MethodPost, path+"/generations", `{}`, cookie, http.StatusForbidden)
		request(http.MethodPost, "/api/passport-revisions/demo/publish", "", cookie, http.StatusForbidden)
		request(http.MethodPut, sharing, `{"shared_with_all":false}`, cookie, http.StatusForbidden)
	}
	request(http.MethodGet, path+"/passport", "", nil, http.StatusUnauthorized)
	store := access.Store{DB: portal.database}
	if role, err := store.ProductRole(t.Context(), "unknown-user", portal.productID); err != nil || role != "" {
		t.Fatalf("unknown user received shared access: role=%q, err=%v", role, err)
	}

	// Individual roles override broad learner access; removing one falls back to it.
	request(http.MethodPut, path+"/members", `{"email":"existing@example.com","role":"lead"}`, portal.admin, http.StatusOK)
	checkMemberships(outsider, access.RoleLead)
	user, err := store.Authenticate(t.Context(), outsider.Value)
	if err != nil {
		t.Fatal(err)
	}
	request(http.MethodDelete, path+"/members/"+user.ID, "", portal.admin, http.StatusNoContent)
	checkMemberships(outsider, access.RoleLearner)

	// Disabling sharing revokes inherited access immediately and keeps individual roles.
	checkProductSharing(t, request(http.MethodPut, sharing, `{"shared_with_all":false}`, portal.admin, http.StatusOK), false)
	for _, cookie := range []*http.Cookie{outsider, newcomer, portal.register("after-disable@example.com")} {
		checkProducts(cookie, false)
		checkMemberships(cookie, "")
		request(http.MethodGet, path+"/passport", "", cookie, http.StatusNotFound)
		request(http.MethodGet, path+"/passport.md", "", cookie, http.StatusNotFound)
	}
	checkMemberships(portal.lead, access.RoleLead)
	checkMemberships(portal.learner, access.RoleLearner)
	request(http.MethodGet, path+"/passport", "", portal.learner, http.StatusOK)
}
