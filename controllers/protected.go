package controllers

import (
	"fmt"
	"net/http"
)

func (c Controller) ProtectedEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("success.")
	}
}