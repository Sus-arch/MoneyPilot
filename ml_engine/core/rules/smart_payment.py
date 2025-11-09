from datetime import datetime


def recommend_smart_payment(client_data: dict) -> dict | None:
    """
    Рекомендует, с какой карты лучше платить.
    Правило: выбрать счёт с наибольшим доступным остатком.
    """
    accounts = client_data.get("accounts", [])
    balances = client_data.get("balances", [])

    if not accounts or not balances:
        return None

    # 🧩 Объединяем балансы по account_id (берём максимум на случай дублей)
    account_balances = {}
    for b in balances:
        acc_id = b.get("account_id")
        amount = float(b.get("amount", 0))
        print(amount)
        currency = b.get("currency", "RUB")

        if acc_id not in account_balances:
            account_balances[acc_id] = {"amount": amount, "currency": currency}
        else:
            # На случай, если пришло несколько InterimAvailable
            account_balances[acc_id]["amount"] = max(account_balances[acc_id]["amount"], amount)

    # 🔍 Находим счёт с максимальным балансом
    if not account_balances:
        return None

    best_account_id, best_data = max(account_balances.items(), key=lambda x: x[1]["amount"])
    best_account = next((a for a in accounts if a["account_id"] == best_account_id), None)

    if not best_account:
        return None

    nickname = best_account.get("nickname") or best_account.get("name") or best_account_id
    balance = best_data["amount"]
    currency = best_data["currency"]

    return {
        "title": "Умный платёж",
        "description": (
            f"Рекомендуем оплатить с карты «{nickname}» — "
            f"на ней самый высокий доступный остаток ({balance:,.2f} {currency})."
        ),
        "category": "payment",
        "priority": "medium",
    }
