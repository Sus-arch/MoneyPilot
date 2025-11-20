import asyncio
from fastapi import FastAPI, Header, Query
from fastapi.middleware.cors import CORSMiddleware
import httpx
from datetime import datetime, timedelta
from dateutil.relativedelta import relativedelta
from prophet import Prophet
import pandas as pd

from core.advisor import generate_advice
from core.can_afford.can_afford_rule import can_afford_rule
from core.can_afford.transactions_to_df import transactions_to_df
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
        print(client_data)
        print(transactions_to_df(client_data["transactions"]))


        recommendation = can_afford_rule(client_data, amount)
        return {"status": "success", "data": recommendation}

    except Exception as e:
        return {"status": "error", "message": str(e)}

@app.get("/predict_spending")
async def predict_spending(authorization: str = Header(...)):
    """Прогнозирует расходы на следующий месяц."""
    try:
        client = GoApiClient(token=authorization)
        accounts = await client.get_accounts()
        transactions = []
        print("test1")
        for acc in accounts:
            t = await client.get_transactions(
                account_id=acc["account_id"],
                bank_code=acc["bank"],
            )
            transactions.extend(t)
            print("test2")

        df = transactions_to_df(transactions)
        print("test3")
        # Обучаем Prophet
        model = Prophet(daily_seasonality=True, weekly_seasonality=True, yearly_seasonality=False)
        model.fit(df)

        # Прогноз на следующий месяц (30 дней)
        future = model.make_future_dataframe(periods=30)
        forecast = model.predict(future)

        # Берем только будущие значения (последние 30 дней прогноза)
        predicted_period = forecast.tail(30)

        # Суммируем прогноз (yhat), игнорируя возможные отрицательные выбросы (траты не могут быть < 0)
        total_predicted = predicted_period['yhat'].clip(lower=0).sum()

        return {
            "status": "success",
            "forecast": round(float(total_predicted), 2),
            "currency": "RUB",
            "next_month": (datetime.utcnow() + relativedelta(months=1)).strftime("%B %Y")
        }

    except Exception as e:
        return {"status": "error", "message": str(e)}
