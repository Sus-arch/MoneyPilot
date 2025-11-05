from datetime import datetime

def recommend_stress_index(client_data):
    """
    Эвристическая оценка финансового стресса пользователя.
    Основывается на доле расходов и уровне баланса.
    """

    balances = client_data.get("balances", [])
    transactions = client_data.get("transactions", [])

    income = sum(
        float(tx["amount"]["amount"])
        for tx in transactions
        if tx.get("creditDebitIndicator") == "Credit"
    )
    expenses = sum(
        float(tx["amount"]["amount"])
        for tx in transactions
        if tx.get("creditDebitIndicator") == "Debit"
    )
    balance = sum(b["amount"] for b in balances)

    # --- Эвристика: чем выше расходы и ниже баланс, тем выше стресс
    ratio_spending = expenses / income
    ratio_balance = balance / income

    stress_index = 0.6 * ratio_spending + 0.4 * (1 - ratio_balance)
    stress_index = max(0, min(stress_index, 1))

    # --- Интерпретация
    if stress_index < 0.4:
        level, priority = "Низкий", "low"
    elif stress_index < 0.7:
        level, priority = "Средний", "medium"
    else:
        level, priority = "Высокий", "high"

    return {
        "title": f"Финансовый стресс: {level} уровень",
        "description": (
            f"Ваш стресс-индекс составляет {stress_index*100:.0f}%.\n"
            f"Расходы — {ratio_spending*100:.0f}% от дохода, "
            f"баланс покрывает {ratio_balance*100:.0f}% месячного дохода.\n"
            "Рекомендуем снизить траты и увеличить подушку безопасности."
        ),
        "category": "risk",
        "priority": priority,
        "created_at": datetime.utcnow().isoformat() + "Z",
    }
