import { Deferred, Head, Link, router } from "@inertiajs/react";
import AppLayout from "../layouts/AppLayout";
import { date, type Note } from "../types";
interface Props {
  stats?: { note_count: number; pinned_count: number; tag_count: number };
  recent?: Note[];
}
export default function Dashboard({ stats, recent }: Props) {
  return (
    <AppLayout>
      <Head title="Dashboard" />
      <div className="mb-8 flex items-center justify-between gap-4">
        <div>
          <p className="eyebrow mb-2">Your workspace</p>
          <h1 className="page-title">Room for your next idea.</h1>
          <p className="muted mt-3">Capture a thought. Keep what matters.</p>
        </div>
        <Link href="/notes/new" className="button shrink-0">
          + New note
        </Link>
      </div>
      <Deferred
        data="stats"
        fallback={
          <div className="panel animate-pulse py-12 text-stone-400">
            Loading your notebook statistics…
          </div>
        }
        rescue={
          <div className="panel border-amber-300">
            <p>
              Statistics could not be loaded. Your notes are still available.
            </p>
            <Link href="/" className="mt-3 inline-block text-sm underline">
              Try again
            </Link>
          </div>
        }
      >
        <div className="grid gap-4 sm:grid-cols-3">
          {[
            ["All notes", stats?.note_count],
            ["Pinned notes", stats?.pinned_count],
            ["Tags", stats?.tag_count],
          ].map(([name, count]) => (
            <div className="panel" key={name}>
              <p className="muted">{name}</p>
              <p className="mt-3 text-4xl font-semibold tracking-tight">
                {count}
              </p>
            </div>
          ))}
        </div>
      </Deferred>
      <section className="mt-10">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">Recently updated</h2>
          <Link href="/notes" className="text-sm underline underline-offset-4">
            View all notes →
          </Link>
        </div>
        <Deferred
          data="recent"
          fallback={
            <div className="panel muted animate-pulse">
              Loading recent notes…
            </div>
          }
        >
          <div className="panel divide-y divide-stone-100 py-0 dark:divide-stone-800">
            {recent?.length ? (
              recent.map((note) => (
                <div key={note.id} className="flex items-center gap-4 py-5">
                  <Link href={`/notes/${note.id}`} className="min-w-0 flex-1">
                    <h3 className="truncate font-medium">{note.title}</h3>
                    <p className="muted mt-1">{date(note.updated_at)}</p>
                  </Link>
                  <button
                    aria-label={`${note.pinned ? "Unpin" : "Pin"} ${note.title}`}
                    onClick={() =>
                      router.post(
                        `/notes/${note.id}/pin?dashboard=1`,
                        {},
                        { only: ["recent", "stats"], preserveScroll: true },
                      )
                    }
                    className="secondary"
                  >
                    {note.pinned ? "Unpin" : "Pin"}
                  </button>
                </div>
              ))
            ) : (
              <p className="muted py-8">No notes yet. Start with a new note.</p>
            )}
          </div>
        </Deferred>
      </section>
      <div className="mt-8 flex flex-wrap items-center gap-3 text-xs text-stone-500">
        <span>
          Inertia in action: statistics and recent notes load independently;
          pinning merges recent notes by ID.
        </span>
        <Link href="/?failStats=1" className="underline">
          Try deferred rescue
        </Link>
      </div>
    </AppLayout>
  );
}
