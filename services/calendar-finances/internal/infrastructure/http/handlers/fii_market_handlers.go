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
	Ticker   string `json:"ticker"`
	Segment  string `json:"segment"`
}

type FIIMarketItem struct {
	Ticker           string  `json:"ticker"`
	Segment          string  `json:"segment"`
	CurrentPrice     float64 `json:"currentPrice"`
	PriceChange12M   float64 `json:"priceChange12M"`
	DividendYield    float64 `json:"dividendYield"`
	Dividends12M     float64 `json:"dividends12M"`
	LastDividend     float64 `json:"lastDividend"`
	LastDividendDate string  `json:"lastDividendDate"`
}

var marketFIIs = []fiiEntry{
	// Logistica
	{Ticker: "HGLG11", Segment: "Logistica"},
	{Ticker: "BTLG11", Segment: "Logistica"},
	{Ticker: "VILG11", Segment: "Logistica"},
	// Papel (Recebiveis)
	{Ticker: "KNCR11", Segment: "Papel"},
	{Ticker: "KNIP11", Segment: "Papel"},
	{Ticker: "MXRF11", Segment: "Papel"},
	{Ticker: "IRDM11", Segment: "Papel"},
	{Ticker: "CPTS11", Segment: "Papel"},
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
	// Fundo de Fundos
	{Ticker: "BCFF11", Segment: "Fundo de Fundos"},
	// Residencial
	{Ticker: "TGAR11", Segment: "Residencial"},
	// Educacional/Hospitalar
	{Ticker: "NSLU11", Segment: "Educacional"},
	// Outros
	{Ticker: "XPLG11", Segment: "Logistica"},
	{Ticker: "VGIR11", Segment: "Papel"},
	{Ticker: "RZTR11", Segment: "Agro"},
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
					CurrentPrice:     math.Round(data.CurrentPrice*100) / 100,
					PriceChange12M:   math.Round(data.PriceChange12M*100) / 100,
					DividendYield:    math.Round(data.DividendYield*100) / 100,
					Dividends12M:     math.Round(data.Dividends12M*100) / 100,
					LastDividend:     math.Round(data.LastDividend*100) / 100,
					LastDividendDate: data.LastDividendDate,
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": results})
	}
}
