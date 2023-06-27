package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

type ApiError struct {
	Error string `json:"error"`
}

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := echo.New()
	router.GET("/account", s.handleGetAccount)
	router.GET("/account/:id", withJWTAuth(s.handleGetAccountById, s.store))
	router.POST("/account", s.handleCreateAccount)
	router.DELETE("/account/:id", s.handleDeleteAccount)
	router.POST("/transfer", s.handleTransfer)
	router.POST("/login", s.handleLogin)
	log.Println("JSON API server running on port ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleLogin(c echo.Context) error {
	var req LoginRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return err
	}

	acc, err := s.store.GetAccountByNumber(int(req.Number))
	if err != nil {
		WriteJSON(c.Response().Writer, http.StatusNotFound, req)
	}
	if !acc.ValidPassword(req.Password) {
		return fmt.Errorf("not authenticated")
	}

	token, err := createJWT(acc)
	if err != nil {
		return err
	}

	resp := LoginResponse{
		Token: token,
		Number: acc.Number,
	}

	return WriteJSON(c.Response().Writer, http.StatusOK, resp)
}

func (s *APIServer) handleGetAccount(c echo.Context) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	return WriteJSON(c.Response().Writer, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountById(c echo.Context) error {
	id, err := getID(c)
	if err != nil {
		return err
	}
	account, err := s.store.GetAccountByID(id)
	if err != nil {
		return err
	}

	return WriteJSON(c.Response().Writer, http.StatusOK, account)
}

func (s *APIServer) handleCreateAccount(c echo.Context) error {
	createAccountRequest := CreateAccountRequest{}
	if err := json.NewDecoder(c.Request().Body).Decode(&createAccountRequest); err != nil {
		return err
	}

	account, err := NewAccount(createAccountRequest.FirstName, createAccountRequest.LastName, createAccountRequest.Password)
	if err != nil {
		return err
	}
	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	return WriteJSON(c.Response().Writer, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(c echo.Context) error {
	id, err := getID(c)
	if err != nil {
		return err
	}
	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}
	return WriteJSON(c.Response().Writer, http.StatusOK, map[string]int{"deleted": id})
}

func (s *APIServer) handleTransfer(c echo.Context) error {
	transferReq := &TransferRequest{}
	if err := json.NewDecoder(c.Request().Body).Decode(transferReq); err != nil {
		return err
	}
	defer c.Request().Body.Close()
	return WriteJSON(c.Response().Writer, http.StatusOK, transferReq)
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func withJWTAuth(handlerFunc echo.HandlerFunc, s Storage) echo.HandlerFunc {
	return func(c echo.Context) error {
		fmt.Println("calling JWT auth middleware")
		tokenString := c.Request().Header.Get("x-jwt-token")
		token, err := validateJWT(tokenString)
		if err != nil || !token.Valid {
			return WriteJSON(c.Response().Writer, http.StatusForbidden, ApiError{Error: "permission denied"})
		}

		id, err := getID(c)
		if err != nil {
			return err
		}
		a, err := s.GetAccountByID(id)
		if err != nil {
			return err
		}
		claims := token.Claims.(jwt.MapClaims)

		if a.Number != int64(claims["accountNumber"].(float64)) {
			return WriteJSON(c.Response().Writer, http.StatusForbidden, ApiError{Error: "permission denied"})
		}

		return handlerFunc(c)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     15000,
		"accountNumber": account.Number,
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func getID(c echo.Context) (int, error) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return -1, fmt.Errorf("invalid id given %s", idStr)
	}

	return id, nil
}
