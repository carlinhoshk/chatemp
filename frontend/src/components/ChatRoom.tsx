import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { getIdentity, shortHash } from "../lib/identity";
import { getRoom } from "../lib/api";
import { formatCountdown } from "../lib/time";
import { useWebSocket } from "../hooks/useWebSocket";
import MessageBubble from "./MessageBubble";
import type { Message, Room, SelfInfo, WsPayload } from "../types";

type Phase = "loading" | "chat" | "expired" | "invalid";

export default function ChatRoom() {
  const { code = "" } = useParams();
  const identity = useMemo(getIdentity, []);
  const [phase, setPhase] = useState<Phase>("loading");
  const [room, setRoom] = useState<Room | null>(null);
  const [self, setSelf] = useState<SelfInfo | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [users, setUsers] = useState<string[]>([]);
  const [input, setInput] = useState("");
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);
  const [now, setNow] = useState(() => new Date());
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    getRoom(code)
      .then((r) => setRoom(r))
      .catch((e: Error) => {
        if (e.message.includes("410")) setPhase("expired");
        else if (e.message.includes("404")) setPhase("invalid");
        else setError(e.message);
      });
  }, [code]);

  useEffect(() => {
    const iv = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(iv);
  }, []);

  const onMessage = useCallback(
    (p: WsPayload) => {
      switch (p.type) {
        case "welcome":
          setRoom(p.room ?? room);
          setSelf(p.self ?? null);
          setUsers(p.users ?? []);
          setMessages(p.messages ?? []);
          setPhase("chat");
          break;
        case "message":
          if (p.message) {
            setMessages((prev) =>
              prev.some((m) => m.id === p.message!.id) ? prev : [...prev, p.message!],
            );
          }
          break;
        case "users":
          setUsers(p.users ?? []);
          break;
        case "room_expired":
          setPhase("expired");
          break;
        case "error":
          setError(p.error ?? "Erro desconhecido");
          break;
      }
    },
    [room],
  );

  const { connected, send } = useWebSocket(code, identity, onMessage);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages.length]);

  function sendText(e: React.FormEvent) {
    e.preventDefault();
    const content = input.trim();
    if (!content || !connected) return;
    send({ type: "chat", content });
    setInput("");
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(location.href);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      setError("Não foi possível copiar o link");
    }
  }

  if (phase === "expired") {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4 p-6 text-center">
        <div className="text-5xl">⏳</div>
        <h1 className="text-2xl font-semibold">Esta sala expirou</h1>
        <p className="text-slate-400">Todas as mensagens e mídias foram apagadas.</p>
        <Link
          to="/"
          className="rounded-xl bg-teal-500 px-6 py-3 font-semibold text-slate-950 hover:bg-teal-400"
        >
          Criar nova sala
        </Link>
      </div>
    );
  }

  if (phase === "invalid") {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4 p-6 text-center">
        <div className="text-5xl">🔒</div>
        <h1 className="text-2xl font-semibold">Sala não encontrada</h1>
        <p className="text-slate-400">Verifique o código e tente novamente.</p>
        <Link
          to="/"
          className="rounded-xl border border-slate-700 px-6 py-3 font-semibold text-slate-200 hover:border-slate-500"
        >
          Voltar
        </Link>
      </div>
    );
  }

  const isOwn = (m: Message) => m.sender === self?.hash;

  return (
    <div className="flex h-full flex-col">
      <header className="flex items-center gap-3 border-b border-slate-800 bg-slate-900/80 px-4 py-3 backdrop-blur">
        <Link to="/" className="rounded-lg px-1 text-xl text-slate-400 hover:text-slate-200">
          ‹
        </Link>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="font-mono text-sm font-semibold tracking-widest text-teal-300">
              {code}
            </span>
            <span
              className={`h-1.5 w-1.5 rounded-full ${connected ? "bg-emerald-400" : "bg-rose-400"}`}
            />
          </div>
          <div className="text-xs text-slate-400">
            {room ? (
              <span>
                expira em <span className="font-mono text-slate-300">{formatCountdown(room.expires_at, now)}</span>
              </span>
            ) : (
              "conectando..."
            )}
            {users.length > 0 && <span> · {users.length} online</span>}
          </div>
        </div>
        <button
          onClick={copyLink}
          className="rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-medium text-slate-200 hover:border-slate-500"
        >
          {copied ? "Copiado!" : "Copiar link"}
        </button>
      </header>

      <main className="flex-1 space-y-2 overflow-y-auto px-3 py-4">
        {messages.map((m) => (
          <MessageBubble key={m.id} msg={m} isOwn={isOwn(m)} />
        ))}
        <div ref={bottomRef} />
        {error && <p className="text-center text-xs text-rose-400">{error}</p>}
      </main>

      <footer className="border-t border-slate-800 bg-slate-900/80 px-3 py-2 backdrop-blur">
        <form onSubmit={sendText} className="flex items-center gap-2">
          <span className="hidden px-1 text-xs text-slate-500 sm:block">
            {self ? shortHash(self.hash) : ""}
          </span>
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Digite uma mensagem..."
            maxLength={4000}
            className="min-w-0 flex-1 rounded-xl border border-slate-700 bg-slate-800 px-4 py-2.5 outline-none placeholder:text-slate-500 focus:border-teal-500"
          />
          <button
            type="submit"
            disabled={!connected || !input.trim()}
            className="rounded-xl bg-teal-500 px-4 py-2.5 font-semibold text-slate-950 transition hover:bg-teal-400 disabled:opacity-40"
          >
            ➤
          </button>
        </form>
      </footer>
    </div>
  );
}
