def can_afford_rule(client_data: dict, amount: float) -> dict:
    """
    Проверяет, может ли пользователь позволить себе покупку на заданную сумму.
    Возвращает рекомендацию в стандартном формате.
    """
    balances = client_data.get("balances", [])
    print(balances)
    transactions = client_data.get("transactions", [])

    # 💰 Достаём числовые значения из "amount"
    total_balance = sum(b.get("amount", 0) for b in balances)


    # 💸 Разделяем расходы и доходы
    expenses = [
        float(t["amount"]["amount"])
        for t in transactions
        if t.get("creditDebitIndicator") == "Debit"
    ]

    incomes = [
        float(t["amount"]["amount"])
        for t in transactions
        if t.get("creditDebitIndicator") == "Credit"
    ]

    avg_monthly_expenses = sum(expenses) / 6 if expenses else 0
    avg_monthly_income = sum(incomes) / 6 if incomes else 0

    # 🧮 Правило: безопасно тратить ≤ 40% от доступного остатка
    safe_limit = total_balance * 0.4

    if amount <= safe_limit:
        verdict = "Покупка безопасна — вы можете себе это позволить."
        priority = "low"
    elif amount <= total_balance:
        verdict = "Покупка превышает безопасный лимит, но у вас достаточно средств."
        priority = "medium"
    else:
        verdict = "Покупка может привести к дефициту средств."
        priority = "high"

    if avg_monthly_expenses > avg_monthly_income:
        verdict += " Однако ваши расходы превышают доходы — стоит быть осторожнее."

    return {
        "title": "Оценка планируемой покупки",
        "description": (
            f"{verdict}\n"
            f"Баланс: {total_balance:.0f} ₽\n"
            f"Безопасный лимит: {safe_limit:.0f} ₽\n"
            f"Сумма покупки: {amount:.0f} ₽"
        ),
        "category": "affordability",
        "priority": priority,
    }
