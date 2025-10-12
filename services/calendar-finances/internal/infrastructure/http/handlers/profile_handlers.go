package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type ProfileHandlers struct {
	createUseCase *usecases.CreateProfileUseCase
	listUseCase   *usecases.ListProfilesUseCase
	getUseCase    *usecases.GetProfileUseCase
	updateUseCase *usecases.UpdateProfileUseCase
	deleteUseCase *usecases.DeleteProfileUseCase
}

func NewProfileHandlers(
	createUC *usecases.CreateProfileUseCase,
	listUC *usecases.ListProfilesUseCase,
	getUC *usecases.GetProfileUseCase,
	updateUC *usecases.UpdateProfileUseCase,
	deleteUC *usecases.DeleteProfileUseCase,
) *ProfileHandlers {
	return &ProfileHandlers{
		createUseCase: createUC,
		listUseCase:   listUC,
		getUseCase:    getUC,
		updateUseCase: updateUC,
		deleteUseCase: deleteUC,
	}
}

// Create handles POST /api/v1/profiles
func (h *ProfileHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := h.createUseCase.Execute(input)
	if err != nil {
		if err == usecases.ErrProfileAlreadyExists {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": profile,
	})
}

// List handles GET /api/v1/profiles
func (h *ProfileHandlers) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.listUseCase.Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  profiles,
		"total": len(profiles),
	})
}

// Get handles GET /api/v1/profiles/{id}
func (h *ProfileHandlers) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	profile, err := h.getUseCase.Execute(id)
	if err != nil {
		if err == usecases.ErrProfileNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": profile,
	})
}

// Update handles PUT /api/v1/profiles/{id}
func (h *ProfileHandlers) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input usecases.UpdateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := h.updateUseCase.Execute(id, input)
	if err != nil {
		if err == usecases.ErrProfileNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": profile,
	})
}

// Delete handles DELETE /api/v1/profiles/{id}
func (h *ProfileHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.deleteUseCase.Execute(id); err != nil {
		if err == usecases.ErrProfileNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
