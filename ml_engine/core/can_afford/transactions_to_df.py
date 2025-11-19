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
    for t in transactions:
        print(t)
        amount = float(t["amount"]["amount"])
        print(amount)
        if t.get("creditDebitIndicator") == "Credit":
            rows.append({"ds": pd.to_datetime(t["bookingDateTime"]), "y": amount})
        else:
            continue
        print(amount)


    df = pd.DataFrame(rows)
    df = df.sort_values("ds").reset_index(drop=True)
    df["ds"] = pd.to_datetime(df["ds"]).dt.tz_localize(None)
    print(df)
    daily = df.groupby("ds")["amount"].sum().reset_index()
    print(daily)
    daily.columns = ["ds", "y"]

    return daily
