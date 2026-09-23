package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
)

// maxSyncBody caps what the endpoint will read. The payload is a handful of fields;
// anything larger is a mistake or an attempt to make the server allocate.
const maxSyncBody = 64 << 10

type ContractHandlers struct {
	syncUC *usecases.SyncContractUseCase
}

func NewContractHandlers(syncUC *usecases.SyncContractUseCase) *ContractHandlers {
	return &ContractHandlers{syncUC: syncUC}
}

// Sync handles POST /api/v1/contracts/sync.
//
// The sender transmits COMPLETE deal state on every relevant change — value,
// currency or status — and treats any non-2xx as a log line rather than an error to
// show a user. So the status code is the whole protocol: 400 means "this payload
// will never work, stop sending it", 500 means "try again", and 200 means stored —
// including the case where the delivery was correctly recognised as old and dropped.
func (h *ContractHandlers) Sync(w http.ResponseWriter, r *http.Request) {
	var input usecases.SyncContractInput
	if err := json.NewDecoder(io.LimitReader(r.Body, maxSyncBody)).Decode(&input); err != nil {
		http.Error(w, `{"error":"malformed json body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.syncUC.Execute(input)
	if errors.Is(err, usecases.ErrInvalidContractSync) {
		respondJSONStatus(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err != nil {
		// Deliberately not echoed to the caller: this is our failure, and the detail
		// belongs in our logs, not in a response to another system.
		respondJSONStatus(w, http.StatusInternalServerError,
			map[string]any{"error": "could not store the contract"})
		return
	}

	respondJSON(w, map[string]any{"data": result})
}

func respondJSONStatus(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
