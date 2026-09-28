import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { getIdentity, shortHash } from "../lib/identity";
import { getRoom, uploadMedia } from "../lib/api";
import { formatCountdown } from "../lib/time";
import { useWebSocket } from "../hooks/useWebSocket";
import MessageBubble from "./MessageBubble";
import MediaViewer from "./MediaViewer";
import MediaPreview from "./MediaPreview";
import { compressImage } from "../lib/image";
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
  const [uploading, setUploading] = useState(false);
  const [ephemeral, setEphemeral] = useState(false);
  const [viewer, setViewer] = useState<Message | null>(null);
  const [viewedIds, setViewedIds] = useState<Set<string>>(() => new Set());
  const bottomRef = useRef<HTMLDivElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const cameraRef = useRef<HTMLInputElement>(null);
  const [pending, setPending] = useState<{ file: File; source: "camera" | "file" } | null>(null);

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
        case "media_deleted":
          setMessages((prev) => prev.filter((m) => m.id !== p.message_id));
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
    if (navigator.share && matchMedia("(pointer: coarse)").matches) {
      try {
        await navigator.share({ title: "ChatTemp", text: `Entre na sala ${code}`, url: location.href });
      } catch {
        /* usuário cancelou o compartilhamento */
      }
      return;
    }
    try {
      await navigator.clipboard.writeText(location.href);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      setError("Não foi possível copiar o link");
    }
  }

  async function onFileSelected(
    e: React.ChangeEvent<HTMLInputElement>,
    source: "camera" | "file",
  ) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setError("");
    setUploading(true);
    try {
      setPending({ file: await compressImage(file), source });
    } finally {
      setUploading(false);
    }
  }

  function retake() {
    const source = pending?.source;
    setPending(null);
    setError("");
    (source === "camera" ? cameraRef : fileRef).current?.click();
  }

  async function sendPending() {
    if (!pending || !connected) return;
    setUploading(true);
    setError("");
    try {
      const msg = await uploadMedia(code, pending.file, ephemeral, identity);
      send({ type: "media", message_id: msg.id });
      setPending(null);
      if (ephemeral) setEphemeral(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha no upload");
    } finally {
      setUploading(false);
    }
  }

  function onViewed(msg: Message) {
    if (!msg.is_ephemeral || viewedIds.has(msg.id)) return;
    setViewedIds((prev) => new Set(prev).add(msg.id));
    send({ type: "viewed", message_id: msg.id });
  }

  const isConsumed = (m: Message) => m.viewed || viewedIds.has(m.id);

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
      <header className="flex items-center gap-3 border-b border-slate-800 bg-slate-900/80 px-4 py-3 pt-safe-3 backdrop-blur">
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
          {copied ? "Copiado!" : "Convidar"}
        </button>
      </header>

      <main className="flex-1 space-y-2 overflow-y-auto overscroll-contain px-3 py-4">
        {messages.map((m) => (
          <MessageBubble
            key={m.id}
            msg={m}
            isOwn={isOwn(m)}
            isConsumed={isConsumed(m)}
            onOpen={setViewer}
          />
        ))}
        <div ref={bottomRef} />
        {error && <p className="text-center text-xs text-rose-400">{error}</p>}
      </main>

      <footer className="border-t border-slate-800 bg-slate-900/80 px-2 py-2 pb-safe-2 backdrop-blur sm:px-3">
        <form onSubmit={sendText} className="flex items-center gap-1.5 sm:gap-2">
          <input
            ref={fileRef}
            type="file"
            accept="image/*,video/*"
            className="hidden"
            onChange={(e) => onFileSelected(e, "file")}
          />
          <input
            ref={cameraRef}
            type="file"
            accept="image/*"
            capture="environment"
            className="hidden"
            onChange={(e) => onFileSelected(e, "camera")}
          />
          <button
            type="button"
            onClick={() => setEphemeral((v) => !v)}
            title="Mídia que desaparece após ser vista"
            className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border text-lg transition ${
              ephemeral
                ? "border-rose-500 bg-rose-500/20 text-rose-300"
                : "border-slate-700 text-slate-400 hover:border-slate-500"
            }`}
          >
            🔥
          </button>
          <button
            type="button"
            onClick={() => fileRef.current?.click()}
            disabled={!connected || uploading}
            title="Enviar foto ou vídeo da galeria"
            aria-label="Anexar da galeria"
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-slate-700 text-lg text-slate-300 transition hover:border-slate-500 disabled:opacity-40"
          >
            {uploading ? "⏳" : "📎"}
          </button>
          <button
            type="button"
            onClick={() => cameraRef.current?.click()}
            disabled={!connected || uploading}
            title="Tirar foto"
            aria-label="Abrir câmera"
            className="hidden h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-slate-700 text-lg text-slate-300 transition hover:border-slate-500 disabled:opacity-40 pointer-coarse:flex"
          >
            📷
          </button>
          <span className="hidden px-1 text-xs text-slate-500 sm:block">
            {self ? shortHash(self.hash) : ""}
          </span>
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Mensagem..."
            maxLength={4000}
            enterKeyHint="send"
            className="h-11 min-w-0 flex-1 rounded-xl border border-slate-700 bg-slate-800 px-3 text-base outline-none placeholder:text-slate-500 focus:border-teal-500 sm:px-4"
          />
          <button
            type="submit"
            disabled={!connected || !input.trim()}
            aria-label="Enviar"
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-teal-500 font-semibold text-slate-950 transition hover:bg-teal-400 disabled:opacity-40"
          >
            ➤
          </button>
        </form>
      </footer>

      {pending && (
        <MediaPreview
          file={pending.file}
          ephemeral={ephemeral}
          sending={uploading}
          error={error}
          retakeLabel={pending.source === "camera" ? "Tirar outra" : "Escolher outra"}
          onToggleEphemeral={() => setEphemeral((v) => !v)}
          onRetake={retake}
          onCancel={() => {
            setPending(null);
            setError("");
          }}
          onSend={sendPending}
        />
      )}

      {viewer && (
        <MediaViewer
          message={viewer}
          onClose={() => setViewer(null)}
          onViewed={onViewed}
        />
      )}
    </div>
  );
}
