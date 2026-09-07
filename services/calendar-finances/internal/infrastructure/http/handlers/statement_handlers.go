package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/brunovieira/calendar-finances/internal/application/usecases"
	"github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
	"github.com/brunovieira/calendar-finances/internal/domain/statement"
)

// StatementHandlers is the door the statement comes in through.
type StatementHandlers struct {
	accounts bankaccount.Repository
	importUC *usecases.ImportStatementUseCase
}

func NewStatementHandlers(accounts bankaccount.Repository, importUC *usecases.ImportStatementUseCase) *StatementHandlers {
	return &StatementHandlers{accounts: accounts, importUC: importUC}
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

	result, err := h.importUC.Execute(usecases.ImportStatementInput{
		AccountID:       account.ID,
		Provider:        provider,
		AccountKind:     accountKindOf(account),
		AccountCurrency: account.Currency,
		Payload:         body.Payload,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
