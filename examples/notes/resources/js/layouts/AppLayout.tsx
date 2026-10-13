import { Link, router, usePage } from "@inertiajs/react";
import { useEffect, type ReactNode } from "react";
import type { SharedProps } from "../types";
export function Flash() {
  const { flash } = usePage();
  const message = typeof flash.message === "string" ? flash.message : "";
  return message ? (
    <div
      role="status"
      className="mb-6 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900"
    >
      {message}
    </div>
  ) : null;
}
export default function AppLayout({ children }: { children: ReactNode }) {
  const { props, url } = usePage<SharedProps>();
  useEffect(() => {
    const theme = props.theme ?? "system";
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () =>
      document.documentElement.classList.toggle(
        "dark",
        theme === "dark" || (theme === "system" && media.matches),
      );
    apply();
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [props.theme]);
  return (
    <>
      <header className="border-b border-stone-200 bg-white dark:border-stone-800 dark:bg-stone-900">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center gap-6 px-6 py-5">
          <Link href="/" className="text-xl font-bold tracking-tight">
            ▤ {props.appName ?? "Notes"}
          </Link>
          <nav className="flex gap-5 text-sm" aria-label="Main navigation">
            {[
              ["/", "Dashboard"],
              ["/notes", "Notes"],
              ["/settings", "Settings"],
            ].map(([href, name]) => (
              <Link
                key={href}
                href={href}
                className={
                  (
                    href === "/"
                      ? url.split("?")[0] === "/"
                      : url.startsWith(href)
                  )
                    ? "font-semibold underline decoration-amber-500 decoration-2 underline-offset-8"
                    : "text-stone-500 hover:text-stone-900 dark:hover:text-white"
                }
              >
                {name}
              </Link>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-4">
            <span className="muted hidden sm:block">{props.auth?.name}</span>
            <button
              onClick={() => router.post("/logout")}
              className="text-sm text-stone-500 hover:underline"
            >
              Log out
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-6 py-10">
        <Flash />
        {children}
      </main>
      <footer className="mx-auto max-w-5xl px-6 pb-8 text-xs text-stone-400">
        A quiet place for your ideas. Built with Inertia + Echo.
      </footer>
    </>
  );
}
