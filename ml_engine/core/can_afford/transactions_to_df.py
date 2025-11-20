import pandas as pd
from typing import List, Dict


def transactions_to_df(transactions: List[Dict]) -> pd.DataFrame:
    """
    Преобразует список транзакций в DataFrame для Prophet.


    Возвращает rowsFrame с колонками:
    - ds: дата транзакции
    - y: сумма расхода (отрицательные значения для расходов, положительные для доходов)
    """
    rows = []
    print("Кол-во транзакций: ", len(transactions))
    print("Данные о расходах о месяцах: ", monthly_expenses(transactions))

    tr = pd.DataFrame(transactions)
    tr.to_csv("/app/exports/transactions.csv", index=False, encoding="utf-8")

    # Список слов-маркеров, которые нужно ИСКЛЮЧИТЬ из прогноза
    # Например: переводы людям, погашение кредитов (если это фиксированная сумма, ее проще прибавить вручную)
    blacklist = ["Перевод клиенту", "Платеж по кредиту"]

    for t in transactions:
        # print(t)
        amount = float(t["amount"]["amount"])
        description = t.get("transactionInformation", "")

        if t.get("creditDebitIndicator") != "Debit":
            continue

        # Пропускаем переводы и кредиты
        # if any(word in description for word in blacklist):
        #     continue

        # Пропускаем аномально большие траты (например, > 30 000 за раз), если это не регулярная история
        if abs(amount) > 30000:
            continue

        dt = t.get("valueDateTime") or t.get("bookingDateTime")
        dt = pd.to_datetime(dt).normalize()

        rows.append({"ds": dt, "y": abs(amount)})

    df = pd.DataFrame(rows)
    df.to_csv("/app/exports/output1.csv", index=False, encoding="utf-8")

    df["ds"] = pd.to_datetime(df["ds"]).dt.tz_localize(None)

    # агрегируем по дню
    df = df.groupby("ds")["y"].sum().reset_index()

    # Если в какой-то день не было трат, Prophet должен знать, что траты были 0, а не "нет данных"
    full_range = pd.date_range(start=df['ds'].min(), end=df['ds'].max())
    df = df.set_index('ds').reindex(full_range, fill_value=0).reset_index()
    df.rename(columns={'index': 'ds'}, inplace=True)

    df.to_csv("/app/exports/output2.csv", index=False, encoding="utf-8")

    return df


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
