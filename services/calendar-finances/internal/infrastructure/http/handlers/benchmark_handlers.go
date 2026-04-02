package handlers

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/brunovieira/calendar-finances/internal/infrastructure/yahoo"
)

type BenchmarkHandler struct {
	yahoo *yahoo.Client
}

func NewBenchmarkHandler(yahoo *yahoo.Client) *BenchmarkHandler {
	return &BenchmarkHandler{yahoo: yahoo}
}

type BenchmarkResponse struct {
	CDI      float64  `json:"cdi"`
	Ibovespa *float64 `json:"ibovespa"`
	Poupanca float64  `json:"poupanca"`
	IFIX     *float64 `json:"ifix"`
	Days     int      `json:"days"`
}

func (h *BenchmarkHandler) GetReturns() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fromStr := r.URL.Query().Get("from")
		if fromStr == "" {
			http.Error(w, "from parameter required", http.StatusBadRequest)
			return
		}

		from, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			http.Error(w, "invalid from date format (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		to := time.Now()
		days := to.Sub(from).Hours() / 24

		// CDI ≈ Selic (14.25% a.a. as of Mar 2025, compounding daily over 252 biz days)
		selicRate := 14.25
		cdiReturn := (math.Pow(1+selicRate/100, days/365) - 1) * 100

		// Poupança: 0.5%/month + TR when Selic > 8.5%
		months := days / 30.44
		poupancaReturn := (math.Pow(1.005, months) - 1) * 100

		resp := BenchmarkResponse{
			CDI:      math.Round(cdiReturn*100) / 100,
			Poupanca: math.Round(poupancaReturn*100) / 100,
			Days:     int(days),
		}

		// IBOV (^BVSP)
		if ibov, err := h.yahoo.GetReturn("%5EBVSP", from, to); err != nil {
			log.Printf("benchmark: IBOV fetch failed: %v", err)
		} else {
			v := math.Round(ibov*100) / 100
			resp.Ibovespa = &v
		}

		// IFIX (FII index via XFIX11 ETF)
		if ifix, err := h.yahoo.GetReturn("XFIX11.SA", from, to); err != nil {
			log.Printf("benchmark: IFIX fetch failed: %v", err)
		} else {
			v := math.Round(ifix*100) / 100
			resp.IFIX = &v
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": resp})
	}
}
