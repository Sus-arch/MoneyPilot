from typing import Dict, Optional

from core.utils.analyze_expenses import analyze_expenses
from core.utils.analyze_income import analyze_income


def recommend_auto_savings(client_data: Dict) -> Optional[Dict]:
    """
    Анализирует финансы и предлагает автонакопление, если есть излишки.
    """
    transactions = client_data.get("transactions", [])
    balances = client_data.get("balances", [])

    print("test1")

    # 1. Считаем средние показатели в месяц
    avg_monthly_income = analyze_income(transactions)
    print("test2")
    avg_monthly_expenses = analyze_expenses(transactions)
    print("test3")
    # Если дохода нет, рекомендовать нечего
    if avg_monthly_income <= 0:
        return None

    # Считаем коэффициент трат (Burn Rate)
    spending_ratio = avg_monthly_expenses / avg_monthly_income

    # Логика рекомендации
    # Если клиент тратит меньше 80% от того, что зарабатывает (ratio < 0.8)
    if spending_ratio < 0.8:
        # Считаем "свободные деньги" (Surplus)
        monthly_surplus = avg_monthly_income - avg_monthly_expenses

        # Рекомендуем откладывать 50% от излишка, но не более 20% от дохода (безопасная стратегия)
        recommended_save_amount = min(monthly_surplus * 0.5, avg_monthly_income * 0.2)

        recommended_save_amount = round(recommended_save_amount / 100) * 100

        if recommended_save_amount < 1000:
            return None  # Слишком маленькая сумма для рекомендации

        return {
            "title": "Ваши деньги могут работать",
            "description": (
                f"Вы тратите всего {spending_ratio * 100:.0f}% от дохода, и у вас остается свободных ~{monthly_surplus:.0f} ₽ в месяц.\n"
                f"Рекомендуем настроить автопополнение копилки на {recommended_save_amount:.0f} ₽ в месяц."
            ),
            "category": "savings",
            "priority": "high",
        }

    return None