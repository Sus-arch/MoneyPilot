from fastapi import FastAPI, Query
import requests
from core.advisor import generate_advice

app = FastAPI(title="FinBalance ML Engine")

GO_API_BASE = "http://go-api:8080"  # URL твоего Go backend'а

@app.get("/analyze")
def analyze(client_id: str = Query(..., description="ID клиента, например team200-1")):
    try:
        # --- Запрашиваем данные из Go ---
        accounts = requests.get(f"{GO_API_BASE}/accounts?client_id={client_id}").json()
        transactions = requests.get(f"{GO_API_BASE}/transactions?client_id={client_id}").json()
        products = requests.get(f"{GO_API_BASE}/products?client_id={client_id}").json()

        # --- Генерируем рекомендации ---
        recommendations = generate_advice({
            "client_id": client_id,
            "accounts": accounts,
            "transactions": transactions,
            "products": products
        })

        return {
            "client_id": client_id,
            "recommendations": recommendations
        }

    except Exception as e:
        return {"error": str(e)}
