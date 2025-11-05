from fastapi import FastAPI, Header
import requests
from datetime import datetime, timedelta
from dateutil.relativedelta import relativedelta

from core.advisor import generate_advice
from services.go_api_client import GoApiClient

app = FastAPI(title="FinBalance ML Engine")

GO_API_BASE = "http://api:8080"

@app.get("/analyze")
def analyze(authorization: str = Header(..., description="Bearer токен с фронтенда")):
    """Получает данные от Go API и возвращает рекомендации."""
    try:
        client = GoApiClient(token=authorization)

        # --- Определяем диапазон последнего месяца ---
        today = datetime.utcnow()
        date_to = today.strftime("%Y-%m-%dT%H:%M:%SZ")
        date_from = (today - relativedelta(months=1)).strftime("%Y-%m-%dT%H:%M:%SZ")

        accounts = client.get_accounts()
        balances = []
        transactions = []

        for account in accounts:
            account_id = account["account_id"]
            bank = account["bank"]
            # Балансы
            balances.extend(client.get_balances(account_id, bank))

            # Транзакции с возможностью фильтрации по дате
            account_transactions = client.get_transactions(
                account_id=account_id,
                bank_code=bank,
                date_from=date_from,
                date_to=date_to
            )
            transactions.extend(account_transactions)

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

    except requests.exceptions.RequestException as e:
        return {
            "status": "error",
            "message": f"Ошибка при обращении к Go API: {str(e)}"
        }
    except Exception as e:
        return {
            "status": "error",
            "message": str(e)
        }