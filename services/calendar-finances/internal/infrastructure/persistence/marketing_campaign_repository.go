package persistence

import (
	"database/sql"
	"errors"

	"github.com/brunovieira/calendar-finances/internal/domain/marketingcampaign"
)

// MarketingCampaignRepository implements marketingcampaign.Repository using PostgreSQL
type MarketingCampaignRepository struct {
	db *sql.DB
}

func NewMarketingCampaignRepository(db *sql.DB) *MarketingCampaignRepository {
	return &MarketingCampaignRepository{db: db}
}

func (r *MarketingCampaignRepository) Create(c *marketingcampaign.MarketingCampaign) error {
	query := `
		INSERT INTO finance.marketing_campaigns (
			id, profile_id, name, platform, start_date, end_date,
			budget, revenue_attributed, leads, conversions, notes, is_active,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.Exec(query,
		c.ID, c.ProfileID, c.Name, c.Platform, c.StartDate, c.EndDate,
		c.Budget, c.RevenueAttributed, c.Leads, c.Conversions, c.Notes, c.IsActive,
		c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *MarketingCampaignRepository) FindByID(id string) (*marketingcampaign.MarketingCampaign, error) {
	query := `
		SELECT id, profile_id, name, platform, start_date, end_date,
			budget, revenue_attributed, leads, conversions, notes, is_active,
			created_at, updated_at
		FROM finance.marketing_campaigns WHERE id = $1
	`
	row := r.db.QueryRow(query, id)
	return scanMarketingCampaign(row)
}

func (r *MarketingCampaignRepository) FindByProfile(profileID string) ([]*marketingcampaign.MarketingCampaign, error) {
	query := `
		SELECT id, profile_id, name, platform, start_date, end_date,
			budget, revenue_attributed, leads, conversions, notes, is_active,
			created_at, updated_at
		FROM finance.marketing_campaigns
		WHERE profile_id = $1
		ORDER BY start_date DESC
	`
	rows, err := r.db.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*marketingcampaign.MarketingCampaign
	for rows.Next() {
		c, err := scanMarketingCampaign(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *MarketingCampaignRepository) Update(c *marketingcampaign.MarketingCampaign) error {
	query := `
		UPDATE finance.marketing_campaigns
		SET name=$2, platform=$3, start_date=$4, end_date=$5,
			budget=$6, revenue_attributed=$7, leads=$8, conversions=$9,
			notes=$10, is_active=$11, updated_at=$12
		WHERE id=$1
	`
	result, err := r.db.Exec(query,
		c.ID, c.Name, c.Platform, c.StartDate, c.EndDate,
		c.Budget, c.RevenueAttributed, c.Leads, c.Conversions,
		c.Notes, c.IsActive, c.UpdatedAt,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("marketing campaign not found")
	}
	return nil
}

func (r *MarketingCampaignRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM finance.marketing_campaigns WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("marketing campaign not found")
	}
	return nil
}

// GetTotalSpent returns the sum of confirmed transaction amounts linked to this campaign
func (r *MarketingCampaignRepository) GetTotalSpent(campaignID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM finance.transactions WHERE campaign_id = $1 AND status = 'CONFIRMED'`,
		campaignID,
	).Scan(&total)
	return total, err
}

type campaignScanner interface {
	Scan(dest ...any) error
}

func scanMarketingCampaign(s campaignScanner) (*marketingcampaign.MarketingCampaign, error) {
	c := &marketingcampaign.MarketingCampaign{}
	var (
		endDate sql.NullTime
		notes   sql.NullString
	)
	err := s.Scan(
		&c.ID, &c.ProfileID, &c.Name, &c.Platform, &c.StartDate, &endDate,
		&c.Budget, &c.RevenueAttributed, &c.Leads, &c.Conversions, &notes, &c.IsActive,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("marketing campaign not found")
		}
		return nil, err
	}
	if endDate.Valid {
		c.EndDate = &endDate.Time
	}
	if notes.Valid {
		c.Notes = &notes.String
	}
	return c, nil
}
