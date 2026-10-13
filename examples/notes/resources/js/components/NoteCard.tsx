import { Link } from "@inertiajs/react";
import { date, type Note } from "../types";
export default function NoteCard({ note }: { note: Note }) {
  return (
    <Link
      href={`/notes/${note.id}`}
      className="panel block transition hover:border-stone-400 hover:shadow-sm"
    >
      <div className="mb-3 flex items-start justify-between gap-3">
        <h2 className="font-semibold">{note.title}</h2>
        {note.pinned && (
          <span className="text-xs text-amber-700 dark:text-amber-400">
            Pinned
          </span>
        )}
      </div>
      {note.body && (
        <p className="muted mb-5 line-clamp-2 whitespace-pre-line">
          {note.body}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-2">
        {(note.tags ?? []).map((tag) => (
          <span
            key={tag}
            className="rounded bg-stone-100 px-2 py-1 text-xs text-stone-600 dark:bg-stone-800 dark:text-stone-300"
          >
            #{tag}
          </span>
        ))}
        <time className="ml-auto text-xs text-stone-400">
          {date(note.updated_at)}
        </time>
      </div>
    </Link>
  );
}
