import pandas as pd
from typing import List, Dict
from datetime import datetime


def analyze_income(transactions: List[Dict]) -> float:
    """
    Рассчитывает средний месячный доход на основе транзакций.

    Возвращает:
    avg_monthly_income (средний доход в месяц)
    """

    total_income = 0.0
    dates = set()  # Инициализация набора для сбора уникальных дат

    for t in transactions:
        amount = float(t.get("amount", {}).get("amount"))

        indicator = t.get("creditDebitIndicator")

        # Получаем дату транзакции и добавляем ее в набор
        raw_dt = t.get("valueDateTime") or t.get("bookingDateTime")
        # Преобразуем в дату без времени для корректного расчета диапазона

        dt = pd.to_datetime(raw_dt).normalize().date()
        dates.add(dt)

        if indicator == "Credit":
            total_income += amount

    # Агрегация доходов
    if not dates:
        return 0.0  # Возвращаем 0, если нет данных

    # Расчет диапазона в днях
    # Считаем количество дней между самой ранней и самой поздней транзакцией
    days_span = (max(dates) - min(dates)).days + 1

    # Переводим дни в месяцы
    months_span = max(1, days_span / 30.0)

    # Расчет среднего дохода
    avg_monthly_income = total_income / months_span

    return avg_monthly_income