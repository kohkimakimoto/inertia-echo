import { Head, Link, router } from "@inertiajs/react";
import AppLayout from "../../layouts/AppLayout";
import { date, type Note } from "../../types";
export default function Show({ note }: { note: Note }) {
  return (
    <AppLayout>
      <Head title={note.title} />
      <Link href="/notes" className="muted hover:underline">
        ← All notes
      </Link>
      <article className="panel mt-5">
        <div className="mb-6 flex flex-wrap items-center gap-3">
          <span className="eyebrow">
            {note.pinned ? "Pinned note" : "Personal note"}
          </span>
          <span className="muted ml-auto">Updated {date(note.updated_at)}</span>
        </div>
        <h1 className="page-title break-words">{note.title}</h1>
        <div className="mt-4 flex flex-wrap gap-2">
          {note.tags.map((tag) => (
            <Link
              key={tag}
              href={`/notes?tag=${encodeURIComponent(tag)}`}
              className="rounded bg-stone-100 px-2 py-1 text-xs text-stone-600 dark:bg-stone-800 dark:text-stone-300"
            >
              #{tag}
            </Link>
          ))}
        </div>
        <div className="mt-6 flex flex-wrap gap-3">
          <Link href={`/notes/${note.id}/edit`} className="button">
            Edit note
          </Link>
          <button
            className="secondary"
            onClick={() =>
              router.post(`/notes/${note.id}/pin`, {}, { preserveScroll: true })
            }
          >
            {note.pinned ? "Unpin" : "Pin note"}
          </button>
          <a href="#body" className="secondary">
            Jump to text ↓
          </a>
          <button
            className="ml-auto rounded-lg px-4 py-2 text-sm text-red-600 hover:bg-red-50"
            onClick={() => {
              if (window.confirm("Delete this note? This cannot be undone."))
                router.post(`/notes/${note.id}/delete`);
            }}
          >
            Delete note
          </button>
        </div>
        <div
          id="body"
          className="mt-8 scroll-mt-8 border-t border-stone-200 pt-8 dark:border-stone-800"
        >
          <p className="whitespace-pre-wrap break-words leading-8">
            {note.body || "This note is a blank page."}
          </p>
        </div>
      </article>
      <aside className="mt-6 rounded-xl border border-dashed border-stone-300 p-5 dark:border-stone-700">
        <h2 className="text-sm font-semibold">Take your place with you</h2>
        <p className="muted mb-4 mt-2">
          Jump to the note text first, then open settings. The redirect keeps
          the #body fragment.
        </p>
        <Link
          href={`/notes/${note.id}#body`}
          className="mr-4 text-sm underline"
        >
          1. Jump to #body
        </Link>
        <button
          className="text-sm underline"
          onClick={() => router.post(`/notes/${note.id}/fragment${window.location.hash}`)}
        >
          2. Open settings with fragment
        </button>
      </aside>
    </AppLayout>
  );
}
