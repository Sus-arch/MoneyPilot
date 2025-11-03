from datetime import datetime

def recommend_expense_control(transactions):
    """Совет по контролю расходов"""
    # Пример анализа категорий
    transport_sum = sum(t["amount"] for t in transactions if t["category"] == "transport")
    total_sum = sum(t["amount"] for t in transactions)
    ratio = transport_sum / total_sum if total_sum else 0

    if ratio > 0.3:
        return {
            "title": "Снизьте расходы на транспорт",
            "description": "Вы тратите на транспорт больше 30% бюджета. Попробуйте каршеринг или проездной.",
            "category": "transport",
            "priority": "medium",
            "created_at": datetime.utcnow().isoformat() + "Z"
        }