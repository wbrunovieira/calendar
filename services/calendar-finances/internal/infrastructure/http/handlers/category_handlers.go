package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/gorilla/mux"
)

type CategoryHandlers struct {
	createUseCase *usecases.CreateCategoryUseCase
	listUseCase   *usecases.ListCategoriesUseCase
	updateUseCase *usecases.UpdateCategoryUseCase
	deleteUseCase *usecases.DeleteCategoryUseCase
}

func NewCategoryHandlers(
	createUC *usecases.CreateCategoryUseCase,
	listUC *usecases.ListCategoriesUseCase,
	updateUC *usecases.UpdateCategoryUseCase,
	deleteUC *usecases.DeleteCategoryUseCase,
) *CategoryHandlers {
	return &CategoryHandlers{
		createUseCase: createUC,
		listUseCase:   listUC,
		updateUseCase: updateUC,
		deleteUseCase: deleteUC,
	}
}

// List handles GET /api/v1/categories?profileId=...&type=...
func (h *CategoryHandlers) List(w http.ResponseWriter, r *http.Request) {
	profileID := r.URL.Query().Get("profileId")
	categoryType := r.URL.Query().Get("type")
	var typePtr *string
	if categoryType != "" {
		typePtr = &categoryType
	}

	categories, err := h.listUseCase.Execute(usecases.ListCategoriesInput{
		ProfileID: profileID,
		Type:      typePtr,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  categories,
		"total": len(categories),
	})
}

// Create handles POST /api/v1/categories
func (h *CategoryHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var input usecases.CreateCategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	category, err := h.createUseCase.Execute(input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrProfileNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": category,
	})
}

// Update handles PUT /api/v1/categories/{id}
func (h *CategoryHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input usecases.UpdateCategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	category, err := h.updateUseCase.Execute(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecases.ErrCategoryNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": category,
	})
}

// Delete handles DELETE /api/v1/categories/{id}
func (h *CategoryHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := h.deleteUseCase.Execute(id); err != nil {
		status := http.StatusInternalServerError
		if err == usecases.ErrCategoryNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
