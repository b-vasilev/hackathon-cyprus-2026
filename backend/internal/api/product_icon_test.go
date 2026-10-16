package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestProductIconAccess allows only admins and the product's leads to change its icon.
func TestProductIconAccess(t *testing.T) {
	t.Parallel()
	portal := newTestPortal(t)
	icon := "/api/products/" + portal.productID + "/icon"
	svg := func(markup string) string {
		return `{"data":"` + base64.StdEncoding.EncodeToString([]byte(markup)) + `"}`
	}
	safe := svg(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="16"/></svg>`)

	otherLead := portal.register("other-lead@example.com")
	var other struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(portal.request(http.MethodPost, "/api/products", `{"name":"Other","sources":[{"kind":"text","text":"Other"}]}`, portal.admin, http.StatusCreated)), &other); err != nil {
		t.Fatal(err)
	}
	portal.request(http.MethodPut, "/api/products/"+other.ID+"/members", `{"email":"other-lead@example.com","role":"lead"}`, portal.admin, http.StatusOK)

	portal.request(http.MethodPut, icon, safe, nil, http.StatusUnauthorized)
	portal.request(http.MethodPut, icon, safe, portal.learner, http.StatusForbidden)
	portal.request(http.MethodPut, icon, safe, otherLead, http.StatusNotFound)
	portal.request(http.MethodDelete, icon, "", portal.learner, http.StatusForbidden)
	portal.request(http.MethodPut, icon, svg(`<svg onload="alert(1)"></svg>`), portal.lead, http.StatusBadRequest)
	portal.request(http.MethodPut, icon, `{"data":"not base64"}`, portal.lead, http.StatusBadRequest)

	if body := portal.request(http.MethodPut, icon, safe, portal.lead, http.StatusOK); !strings.Contains(body, `"icon_url":"data:image/svg+xml;base64,`) {
		t.Fatal("lead could not set the icon", body)
	}
	// Learners see the icon wherever the product appears.
	if body := portal.request(http.MethodGet, "/api/products", "", portal.learner, http.StatusOK); !strings.Contains(body, `"icon_url":"data:image/svg+xml;base64,`) {
		t.Fatal("learner does not see the product icon", body)
	}
	portal.request(http.MethodPut, icon, safe, portal.admin, http.StatusOK)
	if body := portal.request(http.MethodDelete, icon, "", portal.lead, http.StatusOK); !strings.Contains(body, `"icon_url":""`) || !strings.Contains(body, `"sources":[{"kind":"text"`) {
		t.Fatal("icon was not removed or sources changed", body)
	}
}
