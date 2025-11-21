from core.utils.analyze_income import analyze_income
from core.utils.calculate_forecast import calculate_forecast


def can_afford_rule(client_data: dict, purchase_amount: float) -> dict:
    """
    Проверяет, может ли пользователь позволить себе покупку на заданную сумму.
    Возвращает рекомендацию в стандартном формате.
    """
    balances = client_data.get("balances", [])
    transactions = client_data.get("transactions", [])
    print("test1")
    # Считаем текущий общий баланс (ликвидность)
    total_balance = sum(
        float(b.get("amount", {}))
        for b in balances
    )
    print("test2")
    # Анализируем историю
    avg_monthly_income = analyze_income(transactions)
    forecast_variable, forecast_fixed = calculate_forecast(transactions)
    forecasted_expenses = forecast_variable + forecast_fixed
    print("test3")
    # ПРАВИЛА
    # Проверка на банкротство
    if purchase_amount > total_balance:
        return {
            "can_afford": False,
            "level": "CRITICAL",
            "message": "Недостаточно средств.",
            "details": f"Ваш баланс: {total_balance:,.0f} ₽, а покупка: {purchase_amount:,.0f} ₽"
        }

    # Проверка на Кассовый разрыв
    # После покупки должно остаться денег минимум на месяц жизни (по прогнозу)
    remaining_balance = total_balance - purchase_amount

    # Буфер безопасности: 10% сверх прогноза модели Prophet
    safety_margin = forecasted_expenses * 1.1
    print("test4")
    if remaining_balance < safety_margin:
        return {
            "can_afford": False,
            "level": "WARNING",
            "message": "Рискованно. Остатка может не хватить на жизнь (согласно прогнозу AI).",
            "details": (
                f"После покупки останется: {remaining_balance:,.0f} ₽.\n"
                f"AI-прогноз ваших расходов: ~{forecasted_expenses:,.0f} ₽.\n"
                f"Рекомендуемая подушка: {safety_margin:,.0f} ₽."
            )
        }
    print("test5")
    # R3: Проверка соотношения к доходу (Income Ratio)
    if avg_monthly_income > 0:
        income_ratio = purchase_amount / avg_monthly_income
        if income_ratio > 0.7:
            return {
                "can_afford": True,
                "level": "CAUTION",
                "message": "Вы можете это позволить, но покупка " + (
                    "превышает ваш доход." if income_ratio > 1 else "очень крупная."),
                "details": (
                    f"Цена составляет {income_ratio:.0%} от вашего среднего дохода ({avg_monthly_income:,.0f} ₽).\n"
                    "Убедитесь, что это не импульсивная трата."
                )
            }

    # Если все проверки пройдены
    return {
        "can_afford": True,
        "level": "SUCCESS",
        "message": "Отлично! Вы можете смело совершить эту покупку.",
        "details": (
            f"Свободных денег после покупки: {remaining_balance:,.0f} ₽.\n"
            f"Этого хватит на {remaining_balance / (forecasted_expenses/30):.0f} дн. привычной жизни."
        )
    }