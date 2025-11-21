import asyncio
import os
from fastapi import FastAPI, Header, Query
from fastapi.middleware.cors import CORSMiddleware
import httpx
from datetime import datetime
from dateutil.relativedelta import relativedelta

from core.advisor import generate_advice

from core.utils.calculate_forecast import calculate_forecast
from core.can_afford.can_afford_rule import can_afford_rule
from services.go_api_client import GoApiClient

app = FastAPI(title="FinBalance ML Engine")

# Получаем CORS origins из переменной окружения или используем значение по умолчанию
cors_origins_env = os.getenv("CORS_ORIGINS", "http://localhost:5173")
cors_origins = [origin.strip() for origin in cors_origins_env.split(",")]

app.add_middleware(
    CORSMiddleware,
    allow_origins=cors_origins,  # адрес вашего фронта
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

GO_API_BASE = "http://api:8080"


@app.get("/analyze")
async def analyze(authorization: str = Header(..., description="Bearer токен с фронтенда")):
    """Получает данные от Go API и возвращает рекомендации."""
    try:
        client = GoApiClient(token=authorization)
        
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
        
        # Проверка подписки для фильтрации stress_index
        is_subscribed = await client.check_subscription()
        if not is_subscribed:
            # Фильтруем stress_index (category="risk") для неподписанных пользователей
            recommendations = [rec for rec in recommendations if rec.get("category") != "risk"]

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
        
        # Проверка подписки
        is_subscribed = await client.check_subscription()
        if not is_subscribed:
            return {
                "status": "error",
                "message": "Эта функция доступна только для подписчиков. Оформите подписку для доступа к функции can_afford."
            }

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
        print("test0")
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
        for acc in accounts:
            t = await client.get_transactions(
                account_id=acc["account_id"],
                bank_code=acc["bank"],
            )
            transactions.extend(t)
        print("test0")
        forecast_variable, forecast_fixed = calculate_forecast(transactions)
        total = forecast_variable + forecast_fixed
        print("test9")
        return {
            "status": "success",
            "forecast": round(total, 2),
            "details": {
                "variable_spending": round(forecast_variable, 2),
                "fixed_obligations": round(forecast_fixed, 2)
            },
            "next_month": (datetime.utcnow() + relativedelta(months=1)).strftime("%B %Y")
        }

    except Exception as e:
        return {"status": "error", "message": str(e)}
