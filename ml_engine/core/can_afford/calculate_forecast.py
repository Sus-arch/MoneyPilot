from typing import List, Dict, Tuple
import pandas as pd
from prophet import Prophet

from core.can_afford.process_transactions import process_transactions


def calculate_forecast(transactions: List[Dict]) -> Tuple[float, float]:
    """
    Запускает Prophet для переменных трат и медиану для фиксированных.
    Возвращает общую сумму прогноза на 30 дней.
    """
    if not transactions:
        return 0.0
    print("test1")
    # Обработка данных
    variable_data, fixed_data = process_transactions(transactions)

    # Prophet (Variable)
    forecast_variable = 0.0
    print("test2")
    # Prophet требует хотя бы 2 точки данных (желательно больше)
    if variable_data and len(variable_data) > 2:
        try:
            df_var = pd.DataFrame(variable_data)
            df_var["ds"] = pd.to_datetime(df_var["ds"]).dt.tz_localize(None)

            # Агрегация по дням
            df_var = df_var.groupby("ds")["y"].sum().reset_index()

            # Заполнение пропусков нулями (Логика из вашего кода)
            df_var = df_var.set_index('ds').reindex(
                pd.date_range(start=df_var['ds'].min(), end=df_var['ds'].max()),
                fill_value=0
            ).reset_index().rename(columns={'index': 'ds'})

            # Обучение
            m = Prophet(daily_seasonality=False, weekly_seasonality=True, yearly_seasonality=False)
            m.fit(df_var)

            # Прогноз на 30 дней
            future = m.make_future_dataframe(periods=30)
            forecast = m.predict(future)

            # Берем сумму только за будущие 30 дней
            forecast_variable = forecast.tail(30)['yhat'].clip(lower=0).sum()
        except Exception as e:
            print(f"Prophet error: {e}. Fallback to average.")
            # Fallback: если Prophet упал (мало данных), считаем среднее
            df_avg = pd.DataFrame(variable_data)
            daily_avg = df_avg['y'].sum() / ((df_avg['ds'].max() - df_avg['ds'].min()).days + 1)
            forecast_variable = daily_avg * 30
    elif variable_data:

        # Fallback для очень малых данных
        val = sum(x['y'] for x in variable_data)
        forecast_variable = val  # Грубая оценка, если данных почти нет
    print("test3")
    # Fixed (Median/Max logic)
    forecast_fixed = 0.0
    if fixed_data:
        df_fixed = pd.DataFrame(fixed_data)
        monthly_fixed = df_fixed.groupby("month")["amount"].sum()

        last_month_val = monthly_fixed.iloc[-1]
        median_val = monthly_fixed.median()
        forecast_fixed = max(last_month_val, median_val)

    return round(forecast_variable, 2), round(forecast_fixed, 2)