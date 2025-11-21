import numpy as np
from typing import List

def calculate_dynamic_threshold(amounts: List[float]) -> float:
    """
    Считает порог аномалии методом IQR.
    Возвращает сумму, выше которой трата считается выбросом.
    """
    if not amounts:
        return 100000.0  # Дефолт, если истории нет

    # Берем только ненулевые траты для честной статистики
    clean_amounts = [a for a in amounts if a > 0]

    if len(clean_amounts) < 4:
        # Берем максимум из имеющегося или дефолт.
        return max(max(clean_amounts, default=0) * 1.5, 50000.0)

    # Считаем квартили
    q1 = np.percentile(clean_amounts, 25)
    q3 = np.percentile(clean_amounts, 75)
    iqr = q3 - q1

    # Считаем верхнюю границу (Extreme Outlier)
    # Коэффициент 3.0 берет только ОЧЕНЬ сильные выбросы.
    # Коэффициент 1.5 более строгий (отсечет просто дорогие покупки).
    upper_fence = q3 + (3.0 * iqr)

    # Ставим минимальный порог
    return max(upper_fence, 30000.0)