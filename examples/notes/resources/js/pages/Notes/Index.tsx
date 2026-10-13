import { Head, InfiniteScroll, Link, router } from "@inertiajs/react";
import { useState, type FormEvent } from "react";
import AppLayout from "../../layouts/AppLayout";
import NoteCard from "../../components/NoteCard";
import type { Note } from "../../types";
interface Props {
  notes: { data: Note[] };
  filters: { q: string; tag: string };
  availableTags: string[];
}
export default function Index({ notes, filters, availableTags }: Props) {
  const [q, setQ] = useState(filters.q);
  const [tag, setTag] = useState(filters.tag);
  function search(e: FormEvent) {
    e.preventDefault();
    router.get(
      "/notes",
      { q, tag },
      {
        only: ["notes", "filters", "availableTags"],
        reset: ["notes"],
        preserveState: true,
        preserveScroll: false,
      },
    );
  }
  return (
    <AppLayout>
      <Head title="Notes" />
      <div className="mb-8 flex items-center justify-between">
        <div>
          <p className="eyebrow mb-2">Your notebook</p>
          <h1 className="page-title">All notes</h1>
        </div>
        <Link href="/notes/new" className="button">
          + New note
        </Link>
      </div>
      <form onSubmit={search} className="mb-6 flex flex-wrap items-end gap-3">
        <div className="min-w-48 flex-1">
          <label htmlFor="search">Search notes</label>
          <input
            id="search"
            type="search"
            placeholder="Search titles and text…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <div className="min-w-36">
          <label htmlFor="tag">Tag</label>
          <select id="tag" value={tag} onChange={(e) => setTag(e.target.value)}>
            <option value="">All tags</option>
            {availableTags.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </div>
        <button className="secondary" type="submit">
          Apply filters
        </button>
        {(filters.q || filters.tag) && (
          <Link
            href="/notes"
            onClick={() => {
              setQ("");
              setTag("");
            }}
            className="secondary"
          >
            Clear
          </Link>
        )}
      </form>
      {notes.data.length ? (
        <InfiniteScroll
          data="notes"
          buffer={300}
          onlyNext
          next={({ loading }) =>
            loading ? (
              <p role="status" className="muted py-8 text-center">
                Loading more notes…
              </p>
            ) : null
          }
        >
          <div className="grid gap-4 sm:grid-cols-2">
            {notes.data.map((note) => (
              <NoteCard note={note} key={note.id} />
            ))}
          </div>
        </InfiniteScroll>
      ) : (
        <div className="panel py-16 text-center">
          <h2 className="text-lg font-medium">No notes found</h2>
          <p className="muted mt-2">Try another search or create a new note.</p>
        </div>
      )}
      <p className="muted mt-6 text-center">
        {notes.data.length} notes loaded · Scroll to explore more
      </p>
    </AppLayout>
  );
}
