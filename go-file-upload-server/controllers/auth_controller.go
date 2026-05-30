package controllers

import (
    "encoding/json"
    "net/http"
    "os"
    "time"

    "go-file-upload-server/domain"
    userrepo "go-file-upload-server/repositories/user"
    "go-file-upload-server/services/httpserver"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

type AuthController struct {
    repo        userrepo.PostgresUserRepository
    errorHandler *httpserver.HttpErrorHandler
    jwtSecret   []byte
}

func NewAuthController(repo userrepo.PostgresUserRepository, errorHandler *httpserver.HttpErrorHandler) AuthController {
    secret := os.Getenv("JWT_SECRET")
    return AuthController{
        repo:        repo,
        errorHandler: errorHandler,
        jwtSecret:   []byte(secret),
    }
}

// BeforeAction implements httpserver.Controller.
func (a AuthController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        handler(w, r)
    }
}

// Routes implements httpserver.Controller.
func (a AuthController) Routes() []httpserver.Route {
    return []httpserver.Route{
        {Pattern: "/api/signup", Method: http.MethodPost, Handler: a.HandleSignup},
        {Pattern: "/api/login", Method: http.MethodPost, Handler: a.HandleLogin},
    }
}

type signupRequest struct {
    FirstName string `json:"firstName"`
    LastName  string `json:"lastName"`
    Email     string `json:"email"`
    Password  string `json:"password"`
}

func (a AuthController) HandleSignup(w http.ResponseWriter, r *http.Request) {
    var req signupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }
    if req.Password == "" || req.Email == "" {
        a.errorHandler.HandleError(http.StatusBadRequest, w, http.ErrMissingFile)
        return
    }

    hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        a.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }

    user, err := domain.NewUser(req.FirstName, req.LastName, req.Email, string(hashed))
    if err != nil {
        a.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    if err := a.repo.CreateUser(user); err != nil {
        a.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }

    // return created user without password
    user.PasswordHash = ""
    writeJSON(w, http.StatusCreated, user)
}

type loginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (a AuthController) HandleLogin(w http.ResponseWriter, r *http.Request) {
    var req loginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    user, err := a.repo.GetByEmail(req.Email)
    if err != nil {
        a.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    if user == nil {
        a.errorHandler.HandleError(http.StatusUnauthorized, w, http.ErrNoCookie)
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        a.errorHandler.HandleError(http.StatusUnauthorized, w, err)
        return
    }

    // create JWT
    claims := jwt.MapClaims{
        "sub": user.Id,
        "email": user.Email,
        "exp": time.Now().Add(24 * time.Hour).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString(a.jwtSecret)
    if err != nil {
        a.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }

    writeJSON(w, http.StatusOK, map[string]string{"token": signed})
}
