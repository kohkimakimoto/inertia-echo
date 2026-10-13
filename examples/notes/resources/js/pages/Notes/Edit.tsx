import { Form, Head, Link } from "@inertiajs/react";
import AppLayout from "../../layouts/AppLayout";
interface Props {
  note: {
    id?: string;
    title: string;
    body: string;
    tags: string[];
    pinned: boolean;
  };
  errors?: Record<string, string>;
}
export default function Edit({ note, errors = {} }: Props) {
  const editing = !!note.id;
  return (
    <AppLayout>
      <Head title={editing ? "Edit note" : "New note"} />
      <Link
        href={editing ? `/notes/${note.id}` : "/notes"}
        className="muted hover:underline"
      >
        ← {editing ? "Back to note" : "All notes"}
      </Link>
      <h1 className="page-title mb-8 mt-4">
        {editing ? "Edit your note" : "A fresh page."}
      </h1>
      <Form
        key={note.id ?? "new"}
        action={editing ? `/notes/${note.id}` : "/notes"}
        method="post"
        transform={(data) => ({
          ...data,
          tags: String(data.tags ?? "")
            .split(",")
            .map((tag) => tag.trim())
            .filter(Boolean),
          pinned: data.pinned === "true",
        })}
      >
        {({ processing }) => (
          <div className="panel space-y-6">
            <div>
              <label htmlFor="title">Title</label>
              <input
                id="title"
                name="title"
                defaultValue={note.title}
                placeholder="Give your idea a title"
                required
                maxLength={200}
                autoFocus
              />
              {errors.title && <p className="error">{errors.title}</p>}
            </div>
            <div>
              <label htmlFor="body">Note</label>
              <textarea
                id="body"
                name="body"
                defaultValue={note.body}
                rows={14}
                placeholder="What's on your mind?"
              />
              {errors.body && <p className="error">{errors.body}</p>}
            </div>
            <div>
              <label htmlFor="tags">Tags</label>
              <input
                id="tags"
                name="tags"
                defaultValue={note.tags?.join(", ")}
                placeholder="ideas, work, personal"
              />
              <p className="muted mt-2">
                Separate tags with commas. Tags are saved in lowercase.
              </p>
              {errors.tags && <p className="error">{errors.tags}</p>}
            </div>
            <label className="flex items-center gap-3">
              <input
                type="checkbox"
                name="pinned"
                value="true"
                defaultChecked={note.pinned}
                className="size-4 accent-stone-800"
              />
              Pin this note
            </label>
            <div className="flex gap-3 border-t border-stone-200 pt-6 dark:border-stone-800">
              <button disabled={processing} className="button">
                {processing ? "Saving…" : "Save note"}
              </button>
              <Link
                href={editing ? `/notes/${note.id}` : "/notes"}
                className="secondary"
              >
                Cancel
              </Link>
            </div>
          </div>
        )}
      </Form>
    </AppLayout>
  );
}
