import pandas as pd
import os
from typing import List, Dict

from core.can_afford.dynamic_threshold import calculate_dynamic_threshold


def process_transactions(transactions: List[Dict]) -> pd.DataFrame:
    """
    Преобразует список транзакций в DataFrame для Prophet.


    Возвращает rowsFrame с колонками:
    - ds: дата транзакции
    - y: сумма расхода (отрицательные значения для расходов, положительные для доходов)
    """
    variable_txs = []
    fixed_txs = []

    print("Данные о расходах о месяцах: ", monthly_expenses(transactions))

    tr = pd.DataFrame(transactions)
    # Используем /tmp для временных файлов, так как у пользователя app есть права на запись
    export_dir = "/tmp/exports"
    os.makedirs(export_dir, exist_ok=True)
    tr.to_csv(f"{export_dir}/transactions.csv", index=False, encoding="utf-8")

    # Список слов-маркеров, которые нужно ИСКЛЮЧИТЬ из прогноза
    # Например: переводы людям, погашение кредитов (если это фиксированная сумма, ее проще прибавить вручную)
    # Ключевые слова для разделения расходов

    FIXED_KEYWORDS = ["loan", "credit", "кредит", "ипотек", "mortgage", "rent", "аренд", "subscription", "подписк"]
    BLACKLIST_KEYWORDS = ["перевод", "transfer"]
    TRAVEL_KEYWORDS = ["путешествия", "travel", "hotel", "авиабилеты"]

    all_debit_amounts = []
    for t in transactions:
        if t.get("creditDebitIndicator") == "Debit":
            all_debit_amounts.append(abs(float(t["amount"]["amount"])))

    anomaly_limit = calculate_dynamic_threshold(all_debit_amounts)

    for t in transactions:
        amount = float(t["amount"]["amount"])
        desc = t.get("transactionInformation", "").lower()

        if t.get("creditDebitIndicator") != "Debit":
            continue

        # Пропускаем переводы и кредиты
        if any(word in desc for word in BLACKLIST_KEYWORDS):
            continue

        # Исключаем разовые гигантские траты (например > 300к), если это не кредит
        # Это спасает от случайных покупок машины/квартиры, ломающих прогноз

        dt = t.get("valueDateTime") or t.get("bookingDateTime")
        dt = pd.to_datetime(dt).normalize()

        # Сортируем: Фиксированные vs Переменные
        is_fixed = any(w in desc for w in FIXED_KEYWORDS)
        is_travel = any(w in desc for w in TRAVEL_KEYWORDS)

        if is_travel:
            continue

        elif is_fixed:
            fixed_txs.append({"month": dt.strftime("%Y-%m"), "amount": amount})


        else:
            if amount <= anomaly_limit:
                variable_txs.append({"ds": dt, "y": amount})


    return variable_txs, fixed_txs


def monthly_expenses(transactions: List[Dict]) -> Dict[str, float]:
    rows = []

    for t in transactions:
        amount = float(t["amount"]["amount"])
        indicator = t.get("creditDebitIndicator", "").lower()

        # берем фактическую дату операции
        raw_dt = t.get("valueDateTime") or t.get("bookingDateTime")
        dt = pd.to_datetime(raw_dt)

        # расход → отрицательное число
        if indicator.startswith("deb"):
            y = abs(amount)
        else:
            continue  # доходы пропускаем

        rows.append({"ds": dt, "amount": y})

    df = pd.DataFrame(rows)

    df["month"] = df["ds"].dt.to_period("M").astype(str)

    result = (
        df.groupby("month")["amount"]
        .sum()
        .round(2)
        .to_dict()
    )

    return result
