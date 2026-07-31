import type { Message, Room } from "../types";

async function req<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    let detail = `${res.status} ${res.statusText}`;
    try {
      const data = (await res.json()) as { error?: string };
      if (data?.error) detail = data.error;
    } catch {
      /* keep status text */
    }
    throw new Error(detail);
  }
  return res.json() as Promise<T>;
}

export async function createRoom(identity: string): Promise<Room> {
  return req<Room>("/api/rooms", {
    method: "POST",
    headers: { "X-User": identity },
  });
}

export async function getRoom(code: string): Promise<Room> {
  return req<Room>(`/api/rooms/${encodeURIComponent(code)}`);
}

export async function fetchMessages(code: string): Promise<Message[]> {
  return req<Message[]>(`/api/rooms/${encodeURIComponent(code)}/messages`);
}

export async function uploadMedia(
  code: string,
  file: File,
  ephemeral: boolean,
  identity: string,
): Promise<Message> {
  const fd = new FormData();
  fd.append("file", file);
  return req<Message>(
    `/api/rooms/${encodeURIComponent(code)}/media?ephemeral=${ephemeral}`,
    {
      method: "POST",
      headers: { "X-User": identity },
      body: fd,
    },
  );
}
