from datetime import datetime

from core.rules.entertainment_control import recommend_entertainment_control
from core.rules.smart_payment import recommend_smart_payment
from core.rules.auto_savings import recommend_auto_savings


def generate_advice(client_data: dict):
    """
    Возвращает список рекомендаций в виде массива dict'ов.
    Формат элемента списка:
    {
        "id": int,
        "title": str,
        "description": str,
        "category": str,
        "priority": str,
        "created_at": str (ISO-8601)
    }
    """
    recommendations = []
    '''
    rec = recommend_smart_payment(client_data)
    if rec:
        rec.update({
        "id": len(recommendations) + 1,
        "created_at": datetime.utcnow().isoformat() + "Z"
        })
        recommendations.append(rec)
    '''

    for func in [
        recommend_smart_payment,
        recommend_auto_savings,
        recommend_entertainment_control
    ]:
        try:
            rec = func(client_data)
            if rec:
                rec.update({
                    "id": len(recommendations) + 1,
                    "created_at": datetime.utcnow().isoformat() + "Z"
                })
                recommendations.append(rec)
        except Exception as e:
            print(f"[advisor] Ошибка в {func.__name__}: {e}")


    return recommendations
