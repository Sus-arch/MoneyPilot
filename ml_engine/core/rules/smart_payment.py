from datetime import datetime


def recommend_smart_payment(client_data):
    """
    Рекомендует, с какой карты лучше платить.
    Правило: выбрать счет с наибольшим доступным остатком.
    """
    accounts = client_data.get("accounts", [])
    balances = client_data.get("balances", [])

    if not accounts or not balances:
        return None

    # Соединяем счета и балансы по account_id
    account_balances = []
    for acc in accounts:
        acc_id = acc.get("account_id")
        balance_info = next((b for b in balances if b["account_id"] == acc_id), None)
        if balance_info:
            account_balances.append({
                "account_id": acc_id,
                "nickname": acc.get("nickname"),
                "balance": balance_info["amount"]
            })

    if not account_balances:
        return None

    # Выбираем счет с максимальным балансом
    best_account = max(account_balances, key=lambda c: c["balance"])

    return {
        "title": "Умный платёж",
        "description": f"Рекомендуем оплатить с карты '{best_account['nickname']}' — на ней самый высокий доступный остаток ({best_account['balance']} ₽).",
        "category": "payment",
        "priority": "medium",
    }
