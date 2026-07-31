import type { Message } from "../types";
import { formatTime } from "../lib/time";

interface Props {
  msg: Message;
  isOwn: boolean;
}

export default function MessageBubble({ msg, isOwn }: Props) {
  return (
    <div className={`flex flex-col ${isOwn ? "items-end" : "items-start"}`}>
      {!isOwn && (
        <span className="mb-0.5 px-1 text-[10px] font-medium text-slate-500">
          {msg.sender_short}
        </span>
      )}
      <div
        className={`max-w-[85%] rounded-2xl px-3.5 py-2 text-[15px] leading-relaxed shadow ${
          isOwn
            ? "rounded-br-md bg-teal-500 text-slate-950"
            : "rounded-bl-md bg-slate-800 text-slate-100"
        }`}
      >
        <p className="whitespace-pre-wrap break-words">{msg.body}</p>
        <span
          className={`mt-0.5 block text-right text-[10px] ${
            isOwn ? "text-slate-800" : "text-slate-400"
          }`}
        >
          {formatTime(msg.created_at)}
        </span>
      </div>
    </div>
  );
}
