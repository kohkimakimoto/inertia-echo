import type { ReactNode } from "react";
import { Flash } from "./AppLayout";
export default function GuestLayout({ children }: { children: ReactNode }) {
  return (
    <main className="flex min-h-screen items-center justify-center px-6 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center text-2xl font-bold">▤ Notes</div>
        <Flash />
        <div className="panel p-8">{children}</div>
        <p className="muted mt-6 text-center">Your ideas, all in one place.</p>
      </div>
    </main>
  );
}
