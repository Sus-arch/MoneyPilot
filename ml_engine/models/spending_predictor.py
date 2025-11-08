import pandas as pd
import numpy as np
from datetime import datetime
import lightgbm as lgb
from dateutil.relativedelta import relativedelta
from typing import List, Dict


class SpendingPredictor:
    def __init__(self):
        self.model = None

    def prepare_monthly_data(self, transactions: List[Dict]) -> pd.DataFrame:
        """Агрегирует транзакции по месяцам и рассчитывает траты/доход."""
        df = pd.DataFrame(transactions)

        if df.empty:
            return pd.DataFrame()

        # Конвертируем даты
        df["date"] = pd.to_datetime(df["bookingDateTime"])
        df["month"] = df["date"].dt.to_period("M").dt.to_timestamp()

        # Определяем тип операции
        df["amount"] = df["amount"].astype(float)
        df["is_income"] = df["creditDebitIndicator"].eq("Credit")

        monthly = df.groupby("month").agg(
            income=("amount", lambda x: x[df["is_income"]].sum()),
            expenses=("amount", lambda x: x[~df["is_income"]].sum()),
            num_tx=("amount", "count")
        ).reset_index()

        # Заполняем пропуски нулями
        monthly = monthly.fillna(0)

        # Добавляем признаки лагов
        for lag in [1, 2, 3]:
            monthly[f"lag_{lag}"] = monthly["expenses"].shift(lag)

        monthly["roll_mean_3"] = monthly["expenses"].shift(1).rolling(3).mean()
        monthly["month_num"] = monthly["month"].dt.month
        monthly = monthly.dropna()

        return monthly

    def train(self, df: pd.DataFrame):
        """Обучает модель на данных по месяцам."""
        if df.empty or len(df) < 6:
            print("Недостаточно данных для обучения.")
            return None

        X = df[["lag_1", "lag_2", "lag_3", "roll_mean_3", "month_num"]]
        y = df["expenses"]

        dtrain = lgb.Dataset(X, label=y)
        params = {
            "objective": "regression",
            "metric": "mae",
            "verbosity": -1
        }

        self.model = lgb.train(params, dtrain, num_boost_round=100)
        return self.model

    def predict_next_month(self, df: pd.DataFrame):
        """Делает прогноз расходов на следующий месяц."""
        if self.model is None or df.empty:
            return None

        last_row = df.iloc[-1]
        X_pred = pd.DataFrame([{
            "lag_1": last_row["expenses"],
            "lag_2": last_row["lag_1"],
            "lag_3": last_row["lag_2"],
            "roll_mean_3": df["expenses"].tail(3).mean(),
            "month_num": (last_row["month"] + relativedelta(months=1)).month
        }])

        return float(self.model.predict(X_pred)[0])
