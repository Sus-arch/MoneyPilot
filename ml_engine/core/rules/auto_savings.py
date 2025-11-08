def recommend_auto_savings(client_data):
    balances = client_data.get("balances", [])
    transactions = client_data.get("transactions", [])

    income = 0.0
    expenses = 0.0

    for tx in transactions:
        amount = float(tx.get("amount", {}).get("amount", 0))
        code = tx.get("bankTransactionCode", {}).get("code")

        if (code == "ReceivedDebitTransfer") or (code == "ReceivedCreditTransfer"):
            income += amount
        elif (code == "IssuedDebitTransfer") or (code == "IssuedCreditTransfer"):
            expenses += amount

    spending_ratio = expenses / income if income else 1
    total_balance = sum(float(b.get("amount", 0)) for b in balances)

    if spending_ratio < 0.8:
        monthly_savings = 0.2 * income

        return {
            "title": "Рекомендация по сбережениям",
            "description": (
                f"Вы тратите около {spending_ratio * 100:.0f}% от ежемесячного дохода.\n"
                f"Рекомендуется откладывать хотя бы 20% (~{monthly_savings:,.0f} ₽) на накопления.\n"
                f"Сейчас баланс {total_balance:,.0f} ₽. "
            ),
            "category": "savings",
            "priority": "high",
        }

    return None
