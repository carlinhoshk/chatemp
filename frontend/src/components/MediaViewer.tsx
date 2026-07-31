import type { Message } from "../types";

interface Props {
  message: Message;
  onClose: () => void;
}

export default function MediaViewer({ message, onClose }: Props) {
  const src = `/api/media/${message.id}`;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/95 p-4"
      onClick={onClose}
    >
      <button
        onClick={onClose}
        className="absolute right-4 top-4 flex h-10 w-10 items-center justify-center rounded-full bg-slate-800/80 text-xl text-slate-200 hover:bg-slate-700"
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
