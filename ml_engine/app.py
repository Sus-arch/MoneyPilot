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
from core.can_afford.transactions_to_df import process_transactions
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

        print("test1")
        variable_data, fixed_data = process_transactions(transactions)
        print("test2")
        # Prophet (еда, такси, магазины)
        forecast_variable = 0.0
        if variable_data:
            df_var = pd.DataFrame(variable_data)
            df_var["ds"] = pd.to_datetime(df_var["ds"]).dt.tz_localize(None)
            print("test3")
            # Агрегация по дням
            df_var = df_var.groupby("ds")["y"].sum().reset_index()
            print("test4")
            # Заполнение пропусков нулями
            df_var = df_var.set_index('ds').reindex(
                pd.date_range(start=df_var['ds'].min(), end=df_var['ds'].max()),
                fill_value=0
            ).reset_index().rename(columns={'index': 'ds'})

            print("test5")

            # Обучение
            m = Prophet(daily_seasonality=False, weekly_seasonality=True, yearly_seasonality=False)
            m.fit(df_var)

            # Прогноз на 30 дней
            future = m.make_future_dataframe(periods=30)
            forecast = m.predict(future)

            # Берем сумму только за будущие 30 дней
            forecast_variable = forecast.tail(30)['yhat'].clip(lower=0).sum()

            # Математика для Фиксированных расходов
            forecast_fixed = 0.0
            if fixed_data:
                df_fixed = pd.DataFrame(fixed_data)

                # Считаем общую сумму фиксированных трат за каждый месяц
                monthly_fixed = df_fixed.groupby("month")["amount"].sum()

                # Берем МЕДИАНУ или МАКСИМУМ за последние месяцы
                # Медиана лучше, если были дубли. Максимум лучше, если вы боитесь занизить прогноз.
                # Используем среднее между последним месяцем и медианой для безопасности.
                last_month_val = monthly_fixed.iloc[-1]
                median_val = monthly_fixed.median()
                forecast_fixed = max(last_month_val, median_val)
            print("test6")
            total = forecast_variable + forecast_fixed

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
