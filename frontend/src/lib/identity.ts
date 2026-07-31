const KEY = "chatemp.identity";

export function getIdentity(): string {
  const existing = localStorage.getItem(KEY);
  if (existing) return existing;
  const id = generateId();
  localStorage.setItem(KEY, id);
  return id;
}

export function shortHash(hash: string): string {
  if (hash.length < 9) return hash;
  return `${hash.slice(0, 4)}…${hash.slice(-4)}`;
}

function generateId(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
