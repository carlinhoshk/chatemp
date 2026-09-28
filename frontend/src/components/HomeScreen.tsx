import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { getIdentity } from "../lib/identity";
import { createRoom } from "../lib/api";

export default function HomeScreen() {
  const navigate = useNavigate();
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function onCreate() {
    setBusy(true);
    setError("");
    try {
      const room = await createRoom(getIdentity());
      navigate(`/room/${room.code}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Não foi possível criar a sala");
      setBusy(false);
    }
  }

  function onJoin(e: React.FormEvent) {
    e.preventDefault();
    const c = code.trim().toUpperCase();
    if (c) navigate(`/room/${c}`);
  }

  return (
    <div className="flex min-h-full items-center justify-center p-6">
      <div className="w-full max-w-sm space-y-8 text-center">
        <div>
          <h1 className="text-5xl font-bold tracking-tight">
            Chat<span className="text-teal-400">Temp</span>
          </h1>
          <p className="mt-3 text-slate-400">
            Conversas e mídias temporárias. Sem cadastro, sem histórico.
          </p>
        </div>

        <button
          onClick={onCreate}
          disabled={busy}
          className="w-full rounded-xl bg-teal-500 px-4 py-3 font-semibold text-slate-950 transition hover:bg-teal-400 disabled:opacity-50"
        >
          {busy ? "Criando sala..." : "Criar nova sala"}
        </button>

        <div className="flex items-center gap-3 text-xs uppercase tracking-widest text-slate-600">
          <span className="h-px flex-1 bg-slate-800" />
          ou
          <span className="h-px flex-1 bg-slate-800" />
        </div>

        <form onSubmit={onJoin} className="space-y-3">
          <input
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder="Código da sala"
            autoCapitalize="characters"
            autoComplete="off"
            autoCorrect="off"
            spellCheck={false}
            maxLength={8}
            className="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-center text-lg tracking-[0.3em] outline-none focus:border-teal-500"
          />
          <button
            type="submit"
            disabled={!code.trim()}
            className="w-full rounded-xl border border-slate-700 px-4 py-3 font-semibold text-slate-200 transition hover:border-slate-500 disabled:opacity-40"
          >
            Entrar na sala
          </button>
        </form>

        {error && <p className="text-sm text-rose-400">{error}</p>}
      </div>
    </div>
  );
}
