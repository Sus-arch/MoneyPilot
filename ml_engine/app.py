import asyncio
from fastapi import FastAPI, Header, Query
from fastapi.middleware.cors import CORSMiddleware
import httpx
from datetime import datetime, timedelta
from dateutil.relativedelta import relativedelta

from core.advisor import generate_advice
from core.can_afford.can_afford_rule import can_afford_rule
from models.spending_predictor import SpendingPredictor
from services.go_api_client import GoApiClient

app = FastAPI(title="FinBalance ML Engine")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173"],  # адрес вашего фронта
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

GO_API_BASE = "http://api:8080"


@app.get("/analyze")
async def analyze(authorization: str = Header(..., description="Bearer токен с фронтенда")):
    """Получает данные от Go API и возвращает рекомендации."""
    try:
        async def fetch_account_data(account):
            account_id = account["account_id"]
            bank = account["bank"]
            balances_ = await client.get_balances(account_id, bank)
            transactions_ = await client.get_transactions(
                account_id=account_id,
                bank_code=bank,
                date_from=date_from,
                date_to=date_to
            )
            return balances_, transactions_

        client = GoApiClient(token=authorization)

        # --- Определяем диапазон последнего месяца ---
        today = datetime.utcnow()
        date_to = today.strftime("%Y-%m-%dT%H:%M:%SZ")
        date_from = (today - relativedelta(months=1)).strftime("%Y-%m-%dT%H:%M:%SZ")

        accounts = await client.get_accounts()
        balances = []
        transactions = []

        results = await asyncio.gather(*(fetch_account_data(a) for a in accounts))

        for b, t in results:
            balances.extend(b)
            transactions.extend(t)

        # --- Формируем client_data для advisor ---
        client_data = {
            "accounts": accounts,
            "balances": balances,
            "transactions": transactions
        }

        # --- Генерация рекомендаций ---
        recommendations = generate_advice(client_data)

        return {
            "status": "success",
            "data": recommendations
        }

    except httpx.RequestError as e:
        return {
            "status": "error",
            "message": f"Ошибка при обращении к Go API: {str(e)}"
        }
    except Exception as e:
        return {
            "status": "error",
            "message": str(e)
        }


@app.get("/can_afford")
async def can_afford(
    amount: float = Query(..., description="Сумма предполагаемой покупки"),
    authorization: str = Header(..., description="Bearer токен пользователя")
):
    """
    Проверяет, может ли пользователь позволить себе покупку на указанную сумму.
    """
    try:
        client = GoApiClient(token=authorization)

        # Получаем свежие данные
        accounts = await client.get_accounts()
        balances = []
        transactions = []

        today = datetime.utcnow()
        date_from = today - relativedelta(months=1)

        for acc in accounts:
            b = await client.get_balances(acc["account_id"], acc["bank"])
            t = await client.get_transactions(
                account_id=acc["account_id"],
                bank_code=acc["bank"],
                date_from=date_from.strftime("%Y-%m-%d"),
                date_to=today.strftime("%Y-%m-%d")
            )
            balances.extend(b)
            transactions.extend(t)

        client_data = {"accounts": accounts, "balances": balances, "transactions": transactions}

        recommendation = can_afford_rule(client_data, amount)
        return {"status": "success", "data": recommendation}

    except Exception as e:
        return {"status": "error", "message": str(e)}

@app.get("/predict_spending")
def predict_spending(authorization: str = Header(...)):
    """Прогнозирует расходы на следующий месяц."""
    try:
        client = GoApiClient(token=authorization)
        predictor = SpendingPredictor()

        # Получаем список счетов
        accounts = client.get_accounts()
        all_tx = []

        # Дата диапазон: последние 6 месяцев
        date_to = datetime(2025, 10, 20)
        print(date_to)
        date_from = date_to - relativedelta(months=6)
        print("Flag")
        for acc in accounts:
            print(f"📘 Запрашиваю транзакции для account_id={acc['account_id']}, bank={acc['bank']}", flush=True)
            tx = client.get_transactions(
                account_id=acc["account_id"],
                bank_code=acc["bank"],
                date_from=date_from.strftime("%Y-%m-%d"),
                date_to=date_to.strftime("%Y-%m-%d"),
                limit=200
            )
            print(f"🔹 Получено {len(tx)} транзакций", flush=True)

            all_tx.extend(tx)

        if not all_tx:
            return {"status": "error", "message": "Нет данных для прогнозирования"}

        # Подготовка и прогноз
        df = predictor.prepare_monthly_data(all_tx)
        predictor.train(df)
        next_month_forecast = predictor.predict_next_month(df)

        if next_month_forecast is None:
            return {"status": "error", "message": "Недостаточно данных"}

        return {
            "status": "success",
            "forecast": round(next_month_forecast, 2),
            "currency": "RUB",
            "next_month": (datetime.utcnow() + relativedelta(months=1)).strftime("%B %Y")
        }

    except Exception as e:
        return {"status": "error", "message": str(e)}
