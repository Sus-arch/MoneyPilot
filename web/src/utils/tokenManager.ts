export const tokenManager = {
  get(bankCode: string): string | null {
    return localStorage.getItem(`token_${bankCode}`);
  },
  set(bankCode: string, token: string): void {
    localStorage.setItem(`token_${bankCode}`, token);
  },
  remove(bankCode: string): void {
    localStorage.removeItem(`token_${bankCode}`);
  },
  clearAll(): void {
    Object.keys(localStorage)
      .filter((k) => k.startsWith("token_") || k === "connectedBank")
      .forEach((k) => localStorage.removeItem(k));
  },
};
