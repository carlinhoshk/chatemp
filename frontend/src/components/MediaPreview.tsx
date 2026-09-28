import { useEffect, useState } from "react";

interface Props {
  file: File;
  ephemeral: boolean;
  sending: boolean;
  error?: string;
  retakeLabel: string;
  onToggleEphemeral: () => void;
  onRetake: () => void;
  onCancel: () => void;
  onSend: () => void;
}

export default function MediaPreview({
  file,
  ephemeral,
  sending,
  error,
  retakeLabel,
  onToggleEphemeral,
  onRetake,
  onCancel,
  onSend,
}: Props) {
  const [url, setUrl] = useState("");
  useEffect(() => {
    const u = URL.createObjectURL(file);
    setUrl(u);
    return () => URL.revokeObjectURL(u);
  }, [file]);

  const isVideo = file.type.startsWith("video/");

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-black pt-safe pb-safe">
      <div className="flex items-center justify-between px-4 py-3">
        <button
          type="button"
          onClick={onCancel}
          disabled={sending}
          className="flex h-10 w-10 items-center justify-center rounded-full bg-slate-800/80 text-xl text-slate-200 hover:bg-slate-700 disabled:opacity-40"
          aria-label="Cancelar"
        >
          ✕
        </button>
        <button
          type="button"
          onClick={onToggleEphemeral}
          disabled={sending}
          className={`rounded-full border px-3 py-1.5 text-sm font-medium transition ${
            ephemeral
              ? "border-rose-500 bg-rose-500/20 text-rose-300"
              : "border-slate-700 text-slate-300"
          }`}
        >
          🔥 {ephemeral ? "Some após ver" : "Fica na conversa"}
        </button>
      </div>

      <div className="flex min-h-0 flex-1 items-center justify-center px-2">
        {!url ? null : isVideo ? (
          <video src={url} controls playsInline className="max-h-full max-w-full rounded-lg" />
        ) : (
          <img src={url} alt="Pré-visualização" className="max-h-full max-w-full rounded-lg object-contain" />
        )}
      </div>

      {error && <p className="px-4 pt-3 text-center text-sm text-rose-400">{error}</p>}
      <div className="flex items-center gap-3 px-4 py-4">
        <button
          type="button"
          onClick={onRetake}
          disabled={sending}
          className="flex-1 rounded-xl border border-slate-700 px-4 py-3 font-semibold text-slate-200 transition hover:border-slate-500 disabled:opacity-40"
        >
          {retakeLabel}
        </button>
        <button
          type="button"
          onClick={onSend}
          disabled={sending}
          className="flex-1 rounded-xl bg-teal-500 px-4 py-3 font-semibold text-slate-950 transition hover:bg-teal-400 disabled:opacity-60"
        >
          {sending ? "Enviando..." : "Enviar ➤"}
        </button>
      </div>
    </div>
  );
}
