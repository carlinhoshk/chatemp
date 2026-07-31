import { useEffect, useState } from "react";
import type { Message } from "../types";

interface Props {
  message: Message;
  onClose: () => void;
  onViewed?: (msg: Message) => void;
}

export default function MediaViewer({ message, onClose, onViewed }: Props) {
  const src = `/api/media/${message.id}`;
  const total = message.ttl_seconds ?? 0;
  const [remaining, setRemaining] = useState(total);

  useEffect(() => {
    if (!message.is_ephemeral) return;
    onViewed?.(message);
    const end = Date.now() + total * 1000;
    const iv = window.setInterval(() => {
      const left = Math.max(0, Math.ceil((end - Date.now()) / 1000));
      setRemaining(left);
      if (left <= 0) {
        window.clearInterval(iv);
        onClose();
      }
    }, 200);
    return () => window.clearInterval(iv);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [message.id]);

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-black/95 p-4"
      onClick={message.is_ephemeral ? undefined : onClose}
    >
      {message.is_ephemeral && (
        <>
          <div className="absolute inset-x-0 top-0 h-1.5 bg-slate-800">
            <div
              className="h-full bg-teal-400 transition-all duration-200 ease-linear"
              style={{ width: `${total ? (remaining / total) * 100 : 0}%` }}
            />
          </div>
          <div className="absolute left-4 top-3 flex items-center gap-1.5 rounded-full bg-rose-500 px-3 py-1 text-xs font-semibold text-white">
            🔥 {remaining}s
          </div>
        </>
      )}
      <button
        onClick={onClose}
        className="absolute right-4 top-3 flex h-10 w-10 items-center justify-center rounded-full bg-slate-800/80 text-xl text-slate-200 hover:bg-slate-700"
        aria-label="Fechar"
      >
        ✕
      </button>
      {message.kind === "image" ? (
        <img
          src={src}
          alt=""
          className="max-h-full max-w-full rounded-lg"
          onClick={(e) => e.stopPropagation()}
        />
      ) : (
        <video
          src={src}
          controls
          autoPlay
          playsInline
          className="max-h-full max-w-full rounded-lg"
          onClick={(e) => e.stopPropagation()}
        />
      )}
    </div>
  );
}
