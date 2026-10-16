package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/connectionowner"
)

const sessionCookie = "wayfinder_session"

type userKey struct{}

// productResolver finds the product that owns the requested resource.
type productResolver func(*http.Request) (string, error)

func pathProduct(r *http.Request) (string, error) { return r.PathValue("id"), nil }

func currentUser(r *http.Request) access.User {
	user, _ := r.Context().Value(userKey{}).(access.User)
	return user
}

func (f *Factory) registerAccess(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", f.registerUser)
	mux.HandleFunc("POST /api/auth/login", f.login)
	mux.HandleFunc("POST /api/auth/logout", f.logout)
	mux.HandleFunc("GET /api/auth/me", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		value, err := f.me(r.Context(), currentUser(r))
		respond(w, r, value, err)
	}))
	// Leads pick people from this list when granting product access; only admins change accounts.
	mux.HandleFunc("GET /api/users", f.lead(func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Access.Users(r.Context())
		respond(w, r, value, err)
	}))
	mux.HandleFunc("PATCH /api/users/{id}", f.admin(func(w http.ResponseWriter, r *http.Request) {
		input, ok := decode[access.RoleInput](w, r)
		if !ok {
			return
		}
		value, err := f.Access.SetRole(r.Context(), r.PathValue("id"), input.Role)
		respond(w, r, value, err)
	}))
	mux.HandleFunc("DELETE /api/users/{id}", f.admin(func(w http.ResponseWriter, r *http.Request) {
		noContent(w, r, f.Access.DeleteUser(r.Context(), r.PathValue("id")))
	}))
	mux.HandleFunc("GET /api/products/{id}/members", f.product(true, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Access.Members(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("PUT /api/products/{id}/members", f.product(true, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		input, ok := decode[access.MemberInput](w, r)
		if !ok {
			return
		}
		value, err := f.Access.SetMember(r.Context(), r.PathValue("id"), input)
		respond(w, r, value, err)
	}))
	mux.HandleFunc("DELETE /api/products/{id}/members/{userId}", f.product(true, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		noContent(w, r, f.Access.RemoveMember(r.Context(), r.PathValue("id"), r.PathValue("userId")))
	}))
}

func (f *Factory) me(ctx context.Context, user access.User) (access.Me, error) {
	memberships, err := f.Access.Memberships(ctx, user.ID)
	if err != nil {
		return access.Me{}, fmt.Errorf("load memberships: %w", err)
	}
	return access.Me{User: user, Memberships: memberships}, nil
}

func (f *Factory) registerUser(w http.ResponseWriter, r *http.Request) {
	input, ok := decode[access.Credentials](w, r)
	if !ok {
		return
	}
	user, err := f.Access.Register(r.Context(), input)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	token, err := f.Access.CreateSession(r.Context(), user.ID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	setSessionCookie(w, r, token, int(access.SessionLifetime.Seconds()))
	value, err := f.me(r.Context(), user)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (f *Factory) login(w http.ResponseWriter, r *http.Request) {
	input, ok := decode[access.Credentials](w, r)
	if !ok {
		return
	}
	token, user, err := f.Access.Login(r.Context(), input)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	setSessionCookie(w, r, token, int(access.SessionLifetime.Seconds()))
	value, err := f.me(r.Context(), user)
	respond(w, r, value, err)
}

func (f *Factory) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if err := f.Access.Logout(r.Context(), cookie.Value); err != nil {
			respond(w, r, nil, err)
			return
		}
	}
	setSessionCookie(w, r, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

// signedIn requires a valid session and exposes its user through the request context.
func (f *Factory) signedIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			respond(w, r, nil, access.ErrSession)
			return
		}
		user, err := f.Access.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, access.ErrSession) {
				setSessionCookie(w, r, "", -1)
			}
			respond(w, r, nil, err)
			return
		}
		next(w, r.WithContext(context.WithValue(connectionowner.WithUser(r.Context(), user.ID), userKey{}, user)))
	}
}

// admin restricts a route to administrators.
func (f *Factory) admin(next http.HandlerFunc) http.HandlerFunc {
	return f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r).Role != access.RoleAdmin {
			forbidden(w)
			return
		}
		next(w, r)
	})
}

// lead allows administrators and anyone who leads at least one product.
func (f *Factory) lead(next http.HandlerFunc) http.HandlerFunc {
	return f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if user.Role != access.RoleAdmin {
			leads, err := f.Access.LeadsAny(r.Context(), user.ID)
			if err != nil {
				respond(w, r, nil, err)
				return
			}
			if !leads {
				forbidden(w)
				return
			}
		}
		next(w, r)
	})
}

// product requires a product role: any member may read, only leads may manage.
// Products outside the user's access are reported as missing.
func (f *Factory) product(manage bool, resolve productResolver, next http.HandlerFunc) http.HandlerFunc {
	return f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if user.Role == access.RoleAdmin {
			next(w, r)
			return
		}
		productID, err := resolve(r)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		role, err := f.Access.ProductRole(r.Context(), user.ID, productID)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		switch {
		case role == "":
			problem(w, http.StatusNotFound, "not_found", "The requested item was not found or is disabled.")
		case manage && role != access.RoleLead:
			forbidden(w)
		default:
			next(w, r)
		}
	})
}

func forbidden(w http.ResponseWriter) {
	problem(w, http.StatusForbidden, "forbidden", "You do not have permission for this action.")
}

func noContent(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, value string, maxAge int) {
	//nolint:gosec // Secure follows the request scheme so local HTTP development keeps working.
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: value, Path: "/", HttpOnly: true,
		Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
