from rules.smart_payment import recommend_smart_payment
from rules.auto_savings import recommend_auto_savings
from rules.expense_control import recommend_expense_control

def generate_advice(client_data):
    recommendations = []
    accounts = client_data.get("accounts", [])
    transactions = client_data.get("transactions", [])
    products = client_data.get("products", [])

    for func in [
        recommend_smart_payment,
        recommend_auto_savings,
        recommend_expense_control,
    ]:
        try:
            data = None
            if "account" in func.__name__:
                data = accounts
            elif "transaction" in func.__name__:
                data = transactions
            elif "product" in func.__name__:
                data = products

            rec = func(data)
            if rec:
                rec["id"] = len(recommendations) + 1
                recommendations.append(rec)
        except Exception as e:
            print(f"Error in {func.__name__}: {e}")

    return {"status": "success", "data": recommendations}
