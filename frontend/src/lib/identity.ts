const KEY = "chatemp.identity";

export function getIdentity(): string {
  const existing = localStorage.getItem(KEY);
  if (existing) return existing;
  const id = generateId();
  localStorage.setItem(KEY, id);
  return id;
}

function generateId(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
