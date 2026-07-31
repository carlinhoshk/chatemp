import type { Message } from "../types";
import { formatBytes, formatTime } from "../lib/time";

interface Props {
  msg: Message;
  isOwn: boolean;
  onOpen: (msg: Message) => void;
}

export default function MessageBubble({ msg, isOwn, onOpen }: Props) {
  return (
    <div className={`flex flex-col ${isOwn ? "items-end" : "items-start"}`}>
      {!isOwn && (
        <span className="mb-0.5 px-1 text-[10px] font-medium text-slate-500">
          {msg.sender_short}
        </span>
      )}
      <div
        className={`max-w-[85%] overflow-hidden rounded-2xl shadow ${
          isOwn ? "rounded-br-md" : "rounded-bl-md"
        } ${msg.kind === "text" ? "bg-teal-500 text-slate-950" : "bg-slate-800"}`}
      >
        {msg.kind === "text" ? (
          <div className="px-3.5 py-2 text-[15px] leading-relaxed">
            <p className="whitespace-pre-wrap break-words">{msg.body}</p>
            <span
              className={`mt-0.5 block text-right text-[10px] ${
                isOwn ? "text-slate-800" : "text-slate-400"
              }`}
            >
              {formatTime(msg.created_at)}
            </span>
          </div>
        ) : (
          <button type="button" onClick={() => onOpen(msg)} className="block w-full text-left">
            {msg.kind === "image" ? (
              <img
                src={`/api/media/${msg.id}`}
                alt=""
                loading="lazy"
                className="max-h-72 w-full bg-slate-900 object-cover"
              />
            ) : (
              <video
                src={`/api/media/${msg.id}`}
                muted
                preload="metadata"
                className="max-h-72 w-full bg-slate-900 object-cover"
              />
            )}
            <div
              className={`flex items-center justify-between px-2.5 py-1.5 text-[10px] ${
                isOwn ? "text-slate-500" : "text-slate-400"
              }`}
            >
              <span>{msg.mime?.startsWith("video") ? "🎬 vídeo" : "📷 foto"}</span>
              <span>
                {formatBytes(msg.size_bytes)} · {formatTime(msg.created_at)}
              </span>
            </div>
          </button>
        )}
      </div>
    </div>
  );
}
