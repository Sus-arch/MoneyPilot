from datetime import datetime

def recommend_smart_payment(accounts):
    """Рекомендует, с какой карты оплатить"""
    cards = [a for a in accounts if a["type"] == "card"]
    if not cards:
        return None

    # Пример правила: платить с карты, где остаток > 20% от среднего
    avg_balance = sum(c["balance"] for c in cards) / len(cards)
    best_card = max(cards, key=lambda c: c["balance"])

    if best_card["balance"] > 1.2 * avg_balance:
        return {
            "title": "Умный платёж",
            "description": f"Рекомендуем оплатить с карты {best_card['name']} — остаток выше среднего.",
            "category": "payment",
            "priority": "medium",
            "created_at": datetime.utcnow().isoformat() + "Z"
        }
