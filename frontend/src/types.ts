export type MessageKind = "text" | "image" | "video";

export interface Room {
  id: string;
  code: string;
  created_at: string;
  expires_at: string;
  creator: string;
}

export interface Message {
  id: string;
  sender: string;
  sender_short: string;
  kind: MessageKind;
  body?: string;
  mime?: string;
  size_bytes?: number;
  is_ephemeral: boolean;
  viewed: boolean;
  ttl_seconds?: number;
  created_at: string;
}

export interface SelfInfo {
  hash: string;
  short: string;
}

export interface WsPayload {
  type: "welcome" | "message" | "users" | "media_deleted" | "room_expired" | "error";
  message?: Message;
  message_id?: string;
  room?: Room;
  messages?: Message[];
  users?: string[];
  self?: SelfInfo;
  error?: string;
}

export interface IncomingPayload {
  type: "chat" | "media" | "viewed";
  content?: string;
  message_id?: string;
}
