from datetime import datetime, timedelta

def recommend_entertainment_control(client_data):
    """
    Рекомендация по расходам на развлечения за последние 3 месяца.
    Если расходы превышают рекомендуемый процент от дохода, даём совет.
    """
    transactions = client_data.get("transactions", [])

    if not transactions:
        return None

    # --- Определяем дату 3 месяца назад ---
    three_months_ago = datetime.utcnow() - timedelta(days=90)

    # --- Фильтруем транзакции за последние 3 месяца ---
    recent_transactions = [
        tx for tx in transactions
        if datetime.fromisoformat(tx.get("bookingDateTime")[:19]) >= three_months_ago
    ]

    # --- Считаем доход и расходы на развлечения ---
    total_income = 0.0
    entertainment_expenses = 0.0
    for tx in recent_transactions:
        amount = float(tx.get("amount", {}).get("amount", 0))
        code = tx.get("bankTransactionCode", {}).get("code", "")
        info = tx.get("transactionInformation", "")

        # Доход
        if code in ["ReceivedCreditTransfer", "ReceivedDebitTransfer"]:
            total_income += amount
        # Расход на развлечения (можно использовать эмоджи или категории)
        elif code in ["IssuedDebitTransfer"] and "Развлечения" in info:
            entertainment_expenses += amount

    if total_income == 0:
        return None

    monthly_expense = entertainment_expenses / 3  # средние расходы за месяц
    expense_ratio = monthly_expense / (total_income / 3)  # процент от дохода

    if expense_ratio >= 0.08:
        recommendation_text = (
            f"Ваши расходы на развлечения за последние 3 месяца составили {monthly_expense:,.0f} ₽ в месяц.\n"
            f"Это ~{expense_ratio*100:.0f}% от дохода — выше рекомендуемого уровня 5%.\n"
            f"Рекомендуется сократить расходы на развлечения для накоплений."
        )
    else:
        # recommendation_text = (
        #     f"Ваши расходы на развлечения за последние 3 месяца составили {monthly_expense:,.0f} ₽ в месяц.\n"
        #     f"Это ~{expense_ratio*100:.0f}% от дохода — выше рекомендуемого уровня 5%.\n"
        #     f"Рекомендуется сократить расходы на развлечения для накоплений."
        # )
        return None

    return {
        "title": "Контроль расходов на развлечения",
        "description": recommendation_text,
        "category": "expenses",
        "priority": "medium",
    }
