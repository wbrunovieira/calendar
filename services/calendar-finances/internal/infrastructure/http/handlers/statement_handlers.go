package handlers

import (
	"encoding/json"

	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
	"strings"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/statement"
)

// StatementHandlers is the door the statement comes in through.
type StatementHandlers struct {
	accounts    bankaccount.Repository
	importUC    *usecases.ImportStatementUseCase
	reconcileUC *usecases.ReconcileStatementUseCase
}

func NewStatementHandlers(
	accounts bankaccount.Repository,
	importUC *usecases.ImportStatementUseCase,
	reconcileUC *usecases.ReconcileStatementUseCase,
) *StatementHandlers {
	return &StatementHandlers{accounts: accounts, importUC: importUC, reconcileUC: reconcileUC}
}

type importStatementBody struct {
	// ProviderAccountID is the account's id AT THE PROVIDER. The caller does not get
	// to name an account here: it says which bank account the statement came from, and
	// this service resolves it. Letting the caller pick would put one account's
	// spending on another's statement with nothing to catch it.
	ProviderAccountID string          `json:"providerAccountId"`
	Provider          string          `json:"provider"`
	Payload           json.RawMessage `json:"payload"`
}

// Import handles POST /api/v1/statements/import.
//
// The morning cron pulls a window from the provider and posts it here. Re-posting an
// overlapping window is the normal case, not an error: the unique key on
// (account, provider, external id) decides what is new.
func (h *StatementHandlers) Import(w http.ResponseWriter, r *http.Request) {
	var body importStatementBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.ProviderAccountID) == "" {
		http.Error(w, "providerAccountId is required: it says which account the statement came from", http.StatusBadRequest)
		return
	}
	if len(body.Payload) == 0 {
		http.Error(w, "payload is required", http.StatusBadRequest)
		return
	}

	account, err := h.accounts.FindByProviderAccountID(body.ProviderAccountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if account == nil {
		// Named, so the fix is obvious: map this provider account to an account here.
		// Importing into a guessed account is the one outcome worse than not importing.
		http.Error(w,
			"no account is mapped to provider account "+body.ProviderAccountID+
				": set providerAccountId on the account before importing",
			http.StatusNotFound)
		return
	}

	provider := statement.Provider(strings.ToUpper(strings.TrimSpace(body.Provider)))
	if provider == "" {
		provider = statement.ProviderPluggy
	}
	// An unknown provider is refused rather than accepted. The uniqueness key includes
	// it, so a typo would not fail — it would open a second namespace and import the
	// whole statement again.
	if !provider.Valid() {
		http.Error(w, "unknown provider "+string(provider)+": expected PLUGGY, OFX, CSV or BINANCE", http.StatusBadRequest)
		return
	}

	result, err := h.importUC.Execute(usecases.ImportStatementInput{
		AccountID:       account.ID,
		Provider:        provider,
		AccountKind:     accountKindOf(account),
		AccountCurrency: account.Currency,
		Payload:         body.Payload,
	})
	if err != nil {
		// A write that failed is not bad data. Answering 400 would tell the cron to
		// fix its payload while the database was the thing that was down.
		status := http.StatusBadRequest
		if errors.Is(err, usecases.ErrStatementStorage) {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
		"accountId":   account.ID,
		"accountName": account.Name,
		"inserted":    result.Inserted,
		"updated":     result.Updated,
		"rejected":    result.Rejected,
	}})
}

// accountKindOf decides how to read the provider's sign. A card statement speaks in
// debt and everything else speaks in balance, so the same DEBIT means opposite things.
func accountKindOf(account *bankaccount.BankAccount) statement.AccountKind {
	if account.Type == bankaccount.AccountTypeCreditCard {
		return statement.AccountKindCard
	}
	return statement.AccountKindChecking
}

// Reconcile handles POST /api/v1/bank-accounts/{id}/statement/reconcile.
//
// It links the bank's lines to the entries the system already has, and names the ones
// it will not link. The answer worth reading is `missing`: a charge the bank made that
// the ledger does not have. `ambiguous` is the second: more than one entry fits, and
// choosing between them is not this service's call.
//
// 200 when everything lined up, 409 when there is something to do — missing money,
// an ambiguity, or a forecast the bank has now paid — so the cron can alert on the
// status without parsing the body.
func (h *StatementHandlers) Reconcile(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	// Checked here so a typo is answered 400 and stops, instead of reaching the driver
	// and coming back as a 500 the cron retries forever — with the SQL error text in
	// the body.
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "the account id must be a UUID", http.StatusBadRequest)
		return
	}

	result, err := h.reconcileUC.Execute(id)
	if err != nil {
		// This route takes no body: the only thing the caller supplies is the account
		// id. So there is no bad request to report — either the account is unknown or
		// this service failed, and saying 400 for the second told the cron to stop
		// retrying something a retry would have fixed.
		status := http.StatusInternalServerError
		if errors.Is(err, usecases.ErrBankAccountNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(result.Missing) > 0 || len(result.Ambiguous) > 0 || len(result.ReadyToConfirm) > 0 {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]any{"data": result})
}
