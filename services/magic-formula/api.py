from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import pandas as pd
import os

app = FastAPI(title="Magic Formula API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

EMPRESAS_PATH = os.getenv("EMPRESAS_PATH", "dados_empresas.csv")
IBOV_PATH = os.getenv("IBOV_PATH", "ibov.csv")


def load_and_rank():
    """Load data and compute Magic Formula rankings."""
    df = pd.read_csv(EMPRESAS_PATH)
    df["retorno_mensal"] = df.groupby("ticker")["preco_fechamento_ajustado"].pct_change()
    df["retorno_mensal"] = df.groupby("ticker")["retorno_mensal"].shift(-1)
    df = df[df["volume_negociado"] > 1_000_000]

    df["ranking_ebit_ev"] = df.groupby("data")["ebit_ev"].rank(ascending=False)
    df["ranking_roic"] = df.groupby("data")["roic"].rank(ascending=False)
    df["ranking_final"] = df["ranking_ebit_ev"] + df["ranking_roic"]
    df["ranking_final"] = df.groupby("data")["ranking_final"].rank()
    return df


@app.get("/health")
def health():
    return {"status": "ok"}


@app.get("/api/recommendations")
def recommendations(top: int = 10):
    """Return the top N stocks from the most recent month based on Magic Formula."""
    try:
        df = load_and_rank()
        latest_date = df["data"].max()
        latest = df[df["data"] == latest_date].sort_values("ranking_final").head(top)

        return {
            "data": {
                "date": latest_date,
                "stocks": [
                    {
                        "rank": int(row["ranking_final"]),
                        "ticker": row["ticker"],
                        "price": round(row["preco_fechamento_ajustado"], 2),
                        "ebit_ev": round(row["ebit_ev"], 4),
                        "roic": round(row["roic"], 4),
                        "ranking_ebit_ev": int(row["ranking_ebit_ev"]),
                        "ranking_roic": int(row["ranking_roic"]),
                        "volume": int(row["volume_negociado"]),
                    }
                    for _, row in latest.iterrows()
                ],
            }
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/backtest")
def backtest():
    """Return cumulative returns for Magic Formula vs IBOV."""
    try:
        df = load_and_rank()
        top10 = df[df["ranking_final"] <= 10]

        rentabilidades = top10.groupby("data")["retorno_mensal"].mean().to_frame()
        rentabilidades["magic_formula"] = (rentabilidades["retorno_mensal"] + 1).cumprod() - 1
        rentabilidades = rentabilidades.shift(1).dropna()

        ibov = pd.read_csv(IBOV_PATH)
        retornos_ibov = ibov["fechamento"].pct_change().dropna()
        retorno_acum_ibov = (1 + retornos_ibov).cumprod() - 1

        # Align lengths
        min_len = min(len(rentabilidades), len(retorno_acum_ibov))
        mf_values = rentabilidades["magic_formula"].values[:min_len]
        ibov_values = retorno_acum_ibov.values[:min_len]
        dates = rentabilidades.index[:min_len].tolist()

        return {
            "data": {
                "dates": dates,
                "magic_formula": [round(float(v) * 100, 2) for v in mf_values],
                "ibovespa": [round(float(v) * 100, 2) for v in ibov_values],
            }
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/ranking-history/{ticker}")
def ranking_history(ticker: str):
    """Return ranking history for a specific ticker."""
    try:
        df = load_and_rank()
        ticker_data = df[df["ticker"] == ticker.upper()].sort_values("data")

        if ticker_data.empty:
            raise HTTPException(status_code=404, detail=f"Ticker {ticker} not found")

        return {
            "data": {
                "ticker": ticker.upper(),
                "history": [
                    {
                        "date": row["data"],
                        "rank": int(row["ranking_final"]),
                        "ebit_ev": round(row["ebit_ev"], 4),
                        "roic": round(row["roic"], 4),
                        "price": round(row["preco_fechamento_ajustado"], 2),
                        "in_top10": row["ranking_final"] <= 10,
                    }
                    for _, row in ticker_data.iterrows()
                ],
            }
        }
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
