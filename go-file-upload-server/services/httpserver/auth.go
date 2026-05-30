package httpserver

import (
    "context"
    "fmt"
    "net/http"

    userrepo "go-file-upload-server/repositories/user"
    "go-file-upload-server/domain"

    "github.com/golang-jwt/jwt/v5"
)

type currentUserKeyType string

const currentUserKey currentUserKeyType = "currentUser"

// AuthMiddleware returns a middleware that validates JWT tokens and loads the user into the request context.
func AuthMiddleware(repo *userrepo.PostgresUserRepository, jwtSecret []byte) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            auth := r.Header.Get("Authorization")
            if auth == "" {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            var tokenString string
            fmt.Sscanf(auth, "Bearer %s", &tokenString)
            if tokenString == "" {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
                if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method")
                }
                return jwtSecret, nil
            })
            if err != nil || !token.Valid {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            claims, ok := token.Claims.(jwt.MapClaims)
            if !ok {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            sub, ok := claims["sub"].(string)
            if !ok || sub == "" {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            user, err := repo.GetByID(sub)
            if err != nil || user == nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), currentUserKey, user)
            next(w, r.WithContext(ctx))
        }
    }
}

// GetCurrentUser returns the current authenticated user from the request context, or nil.
func GetCurrentUser(r *http.Request) *domain.User {
    v := r.Context().Value(currentUserKey)
    if v == nil {
        return nil
    }
    if u, ok := v.(*domain.User); ok {
        return u
    }
    return nil
}
