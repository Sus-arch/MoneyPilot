from typing import Optional

import requests

GO_API_BASE = "http://api:8080"


class GoApiClient:
    def __init__(self, token: str):
        self.headers = {"Authorization": token}

    def get_accounts(self):
        resp = requests.get(f"{GO_API_BASE}/api/accounts", headers=self.headers)
        resp.raise_for_status()
        return resp.json().get("accounts", [])

    def get_balances(self, account_id: str, bank_code: str):
        resp = requests.get(
            f"{GO_API_BASE}/api/accounts/{account_id}/balances",
            headers={**self.headers, "X-Bank-Code": bank_code}
        )
        resp.raise_for_status()
        balances = resp.json().get("data", {}).get("balance", [])
        # фильтруем и приводим к нужной структуре
        available_balances = [
            {
                "account_id": b["accountId"],
                "amount": float(b["amount"]["amount"]),
                "currency": b["amount"]["currency"]
            }
            for b in balances
            if b.get("type") == "InterimAvailable"
        ]
        return available_balances

    def get_transactions(
        self,
        account_id: str,
        bank_code: str,
        page: int = 1,
        limit: int = 50,
        date_from: Optional[str] = None,
        date_to: Optional[str] = None
    ):
        params = {
            "page": page,
            "limit": limit
        }
        if date_from:
            params["from"] = date_from
        if date_to:
            params["to"] = date_to

        resp = requests.get(
            f"{GO_API_BASE}/api/accounts/{account_id}/transactions",
            headers={**self.headers, "X-Bank-Code": bank_code},
            params=params
        )
        resp.raise_for_status()
        return resp.json().get("data", {}).get("transaction", [])