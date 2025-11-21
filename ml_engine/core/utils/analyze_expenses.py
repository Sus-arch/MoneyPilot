from typing import List, Dict
import pandas as pd

def analyze_expenses(transactions: List[Dict]) -> float:
    """
    Рассчитывает средний месячный расход.
    Логика зеркальна analyze_income для корректного сравнения.
    """
    total_expenses = 0.0
    dates = set()

    BLACKLIST = ["перевод", "transfer"]

    for t in transactions:
        if t.get("creditDebitIndicator") != "Debit":
            continue

        amount = float(t.get("amount", {}).get("amount", 0))
        desc = (t.get("transactionInformation") or "").lower()

        # Исключаем переводы (это не расход, а перекладывание денег)
        if any(w in desc for w in BLACKLIST):
            continue

        raw_dt = t.get("valueDateTime") or t.get("bookingDateTime")
        if raw_dt:
            dt = pd.to_datetime(raw_dt).normalize().date()
            dates.add(dt)
            total_expenses += abs(amount)

    if not dates:
        return 0.0

    days_span = (max(dates) - min(dates)).days + 1
    months_span = max(1, days_span / 30.0)

    return total_expenses / months_span