from datetime import datetime

def recommend_auto_savings(accounts):
    """Совет перевести излишки на накопительный счёт"""
    savings = [a for a in accounts if a["type"] == "savings"]
    regular = [a for a in accounts if a["type"] == "checking"]

    if not (savings and regular):
        return None

    rich_acc = max(regular, key=lambda a: a["balance"])
    poor_sav = min(savings, key=lambda a: a["balance"])

    if rich_acc["balance"] > 50_000:
        return {
            "title": "Пополнение сбережений",
            "description": f"На счёте {rich_acc['name']} избыточный остаток. "
                           f"Переведите 15 000 ₽ на накопительный {poor_sav['name']}.",
            "category": "savings",
            "priority": "high",
            "created_at": datetime.utcnow().isoformat() + "Z"
        }
