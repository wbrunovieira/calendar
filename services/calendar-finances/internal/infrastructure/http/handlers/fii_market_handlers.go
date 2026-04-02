package handlers

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"

	"github.com/brunovieira/calendar-finances/internal/infrastructure/yahoo"
)

type FIIMarketHandler struct {
	yahoo *yahoo.Client
}

func NewFIIMarketHandler(yahoo *yahoo.Client) *FIIMarketHandler {
	return &FIIMarketHandler{yahoo: yahoo}
}

type fiiEntry struct {
	Ticker  string `json:"ticker"`
	Segment string `json:"segment"`
}

type FIIMonthlyDiv struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
	Yield  float64 `json:"yield"`
}

type FIIMarketItem struct {
	Ticker           string          `json:"ticker"`
	Segment          string          `json:"segment"`
	CurrentPrice     float64         `json:"currentPrice"`
	PriceChange12M   float64         `json:"priceChange12M"`
	DividendYield    float64         `json:"dividendYield"`
	Dividends12M     float64         `json:"dividends12M"`
	LastDividend     float64         `json:"lastDividend"`
	LastDividendDate string          `json:"lastDividendDate"`
	PriceToBook      *float64        `json:"priceToBook"`
	BookValue        *float64        `json:"bookValue"`
	TotalReturn12M   float64         `json:"totalReturn12M"`
	MonthlyDividends []FIIMonthlyDiv `json:"monthlyDividends"`
}

type FIIMarketResponse struct {
	Data     []FIIMarketItem `json:"data"`
	CDIRate  float64         `json:"cdiRate"`
	CDIYield float64         `json:"cdiYield"`
}

var marketFIIs = []fiiEntry{
	// Logistica
	{Ticker: "HGLG11", Segment: "Logistica"},
	{Ticker: "BTLG11", Segment: "Logistica"},
	{Ticker: "VILG11", Segment: "Logistica"},
	{Ticker: "XPLG11", Segment: "Logistica"},
	// Papel (Recebiveis)
	{Ticker: "KNCR11", Segment: "Papel"},
	{Ticker: "KNIP11", Segment: "Papel"},
	{Ticker: "MXRF11", Segment: "Papel"},
	{Ticker: "IRDM11", Segment: "Papel"},
	{Ticker: "CPTS11", Segment: "Papel"},
	{Ticker: "VGIR11", Segment: "Papel"},
	// Shopping
	{Ticker: "XPML11", Segment: "Shopping"},
	{Ticker: "VISC11", Segment: "Shopping"},
	{Ticker: "HSML11", Segment: "Shopping"},
	// Lajes Corporativas
	{Ticker: "KNRI11", Segment: "Lajes Corporativas"},
	{Ticker: "BRCR11", Segment: "Lajes Corporativas"},
	{Ticker: "PVBI11", Segment: "Lajes Corporativas"},
	// Hibrido
	{Ticker: "HGBS11", Segment: "Hibrido"},
	{Ticker: "RBRR11", Segment: "Hibrido"},
	// Agro
	{Ticker: "SNAG11", Segment: "Agro"},
	{Ticker: "RZTR11", Segment: "Agro"},
	// Fundo de Fundos
	{Ticker: "BCFF11", Segment: "Fundo de Fundos"},
	// Residencial
	{Ticker: "TGAR11", Segment: "Residencial"},
	// Educacional/Hospitalar
	{Ticker: "NSLU11", Segment: "Educacional"},
}

func (h *FIIMarketHandler) GetMarketFIIs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			mu      sync.Mutex
			wg      sync.WaitGroup
			results []FIIMarketItem
		)

		sem := make(chan struct{}, 5)

		for _, entry := range marketFIIs {
			wg.Add(1)
			go func(e fiiEntry) {
				defer wg.Done()

				sem <- struct{}{}
				defer func() { <-sem }()

				data, err := h.yahoo.GetFIIData(e.Ticker)
				if err != nil {
					log.Printf("fii-market: failed to fetch %s: %v", e.Ticker, err)
					return
				}

				item := FIIMarketItem{
					Ticker:           e.Ticker,
					Segment:          e.Segment,
					CurrentPrice:     round2(data.CurrentPrice),
					PriceChange12M:   round2(data.PriceChange12M),
					DividendYield:    round2(data.DividendYield),
					Dividends12M:     round2(data.Dividends12M),
					LastDividend:     round2(data.LastDividend),
					LastDividendDate: data.LastDividendDate,
					TotalReturn12M:   round2(data.TotalReturn12M),
				}

				if data.PriceToBook != nil {
					v := round2(*data.PriceToBook)
					item.PriceToBook = &v
				}
				if data.BookValue != nil {
					v := round2(*data.BookValue)
					item.BookValue = &v
				}

				// Monthly dividends
				for _, md := range data.MonthlyDividends {
					item.MonthlyDividends = append(item.MonthlyDividends, FIIMonthlyDiv{
						Month:  md.Month,
						Amount: round2(md.Amount),
						Yield:  round2(md.Yield),
					})
				}

				mu.Lock()
				results = append(results, item)
				mu.Unlock()
			}(entry)
		}

		wg.Wait()

		sort.Slice(results, func(i, j int) bool {
			return results[i].DividendYield > results[j].DividendYield
		})

		// CDI (Selic) rate for comparison — 14.25% a.a. as of Mar 2025
		selicRate := 14.25
		// CDI annual yield (net of 15% IR for > 2 years, for fair comparison with FII which is tax-free)
		cdiYieldNet := selicRate * 0.85

		resp := FIIMarketResponse{
			Data:     results,
			CDIRate:  selicRate,
			CDIYield: round2(cdiYieldNet),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
