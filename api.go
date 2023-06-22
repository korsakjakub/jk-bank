package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type apiFunc func(http.ResponseWriter, *http.Request) error

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
	router.GET("/account/:id", s.handleGetAccountById)
	router.POST("/account", s.handleCreateAccount)
	router.DELETE("/account/:id", s.handleDeleteAccount)
	log.Println("JSON API server running on port ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
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

	account := NewAccount(createAccountRequest.FirstName, createAccountRequest.LastName)
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

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func getID(c echo.Context) (int, error) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return -1, fmt.Errorf("invalid id given %s", idStr)
		}
	return id, nil
}
