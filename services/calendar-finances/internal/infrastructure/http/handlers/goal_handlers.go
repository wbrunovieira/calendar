package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/goal"
	"github.com/gorilla/mux"
)

type GoalHandlers struct {
	service *usecases.GoalsService
}

func NewGoalHandlers(service *usecases.GoalsService) *GoalHandlers {
	return &GoalHandlers{service: service}
}

func (h *GoalHandlers) List(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	if profileID == "" {
		http.Error(w, "profileId is required", http.StatusBadRequest)
		return
	}

	goals, err := h.service.List(profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if goals == nil {
		goals = make([]*goal.Goal, 0)
	}

	respondJSON(w, map[string]interface{}{"data": goals})
}

func (h *GoalHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.GoalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	goal, err := h.service.Create(input)
	if err != nil {
		if err == usecases.ErrInvalidInput {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"data": goal})
}

func (h *GoalHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var input usecases.GoalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	goal, err := h.service.Update(id, input)
	if err != nil {
		if err == usecases.ErrGoalNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{"data": goal})
}

func (h *GoalHandlers) AddAmount(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var input usecases.GoalAddAmountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	goal, err := h.service.AddAmount(id, input)
	if err != nil {
		if err == usecases.ErrGoalNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, map[string]interface{}{"data": goal})
}

func (h *GoalHandlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	goal, err := h.service.UpdateStatus(id, body.Status)
	if err != nil {
		if err == usecases.ErrGoalNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, map[string]interface{}{"data": goal})
}

func (h *GoalHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := h.service.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{"message": "goal deleted"})
}
