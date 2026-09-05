package usecases

import (
	"time"

	"github.com/brunovieira/calendar-finances/internal/domain/recurringtransaction"
)

type ProcessedRecurringResult struct {
	RecurringID   string  `json:"recurringId"`
	Description   string  `json:"description"`
	Amount        float64 `json:"amount"`
	TransactionID string  `json:"transactionId,omitempty"`
	Error         string  `json:"error,omitempty"`
}

type ProcessRecurringOutput struct {
	Processed int                        `json:"processed"`
	Failed    int                        `json:"failed"`
	Results   []ProcessedRecurringResult `json:"results"`
}

type ProcessRecurringTransactionsUseCase struct {
	recurringRepo   recurringtransaction.Repository
	createTxUseCase *CreateTransactionUseCase
}

func NewProcessRecurringTransactionsUseCase(
	recurringRepo recurringtransaction.Repository,
	createTxUseCase *CreateTransactionUseCase,
) *ProcessRecurringTransactionsUseCase {
	return &ProcessRecurringTransactionsUseCase{
		recurringRepo:   recurringRepo,
		createTxUseCase: createTxUseCase,
	}
}

func (uc *ProcessRecurringTransactionsUseCase) Execute() (*ProcessRecurringOutput, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)

	dueRecurrings, err := uc.recurringRepo.FindDueRecurring(now)
	if err != nil {
		return nil, err
	}

	output := &ProcessRecurringOutput{
		Results: make([]ProcessedRecurringResult, 0, len(dueRecurrings)),
	}

	for _, rec := range dueRecurrings {
		result := ProcessedRecurringResult{
			RecurringID: rec.ID,
			Description: rec.Description,
			Amount:      rec.Amount,
		}

		txInput := buildTransactionInput(rec)

		tx, err := uc.createTxUseCase.Execute(txInput)
		if err != nil {
			result.Error = err.Error()
			output.Failed++
			output.Results = append(output.Results, result)
			continue
		}

		result.TransactionID = tx.ID
		output.Processed++

		// Advance next_occurrence: use current next_occurrence + 1 day as reference
		// so CalculateNextOccurrence finds the NEXT date after the current one
		refDate := rec.NextOccurrence.AddDate(0, 0, 1)
		nextOccurrence := rec.CalculateNextOccurrence(refDate)
		if !nextOccurrence.IsZero() {
			rec.NextOccurrence = nextOccurrence
		}
		rec.UpdatedAt = time.Now()
		_ = uc.recurringRepo.Update(rec)

		output.Results = append(output.Results, result)
	}

	return output, nil
}

func buildTransactionInput(rec *recurringtransaction.RecurringTransaction) CreateTransactionInput {
	status := "PLANNED"
	occurredOn := rec.NextOccurrence.Format("2006-01-02")

	var bankAccountID string
	if rec.BankAccountID != nil {
		bankAccountID = *rec.BankAccountID
	}

	return CreateTransactionInput{
		ProfileID:      rec.ProfileID,
		BankAccountID:  bankAccountID,
		CategoryID:     rec.CategoryID,
		Type:           rec.Type,
		Status:         &status,
		Amount:         rec.Amount,
		Currency:       rec.Currency,
		Description:    rec.Description,
		OccurredOn:     occurredOn,
		RecurrenceRule: &rec.RecurrenceRule,
	}
}
