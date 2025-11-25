package handlers

import (
	"encoding/json"
	"net/http"

	"go-microservice/services"
)

type IntegrationHandler struct {
	integrationService *services.IntegrationService
}

func NewIntegrationHandler(integrationService *services.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{
		integrationService: integrationService,
	}
}

func (h *IntegrationHandler) CallExternalAPI(w http.ResponseWriter, r *http.Request) {
	var request struct {
		URL    string            `json:"url"`
		Method string            `json:"method"`
		Data   map[string]string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := h.integrationService.CallExternalAPI(request.URL, request.Method, request.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
