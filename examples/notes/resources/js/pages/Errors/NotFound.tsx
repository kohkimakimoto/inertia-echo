import { Head, Link, usePage } from "@inertiajs/react";
import AppLayout from "../../layouts/AppLayout";
import GuestLayout from "../../layouts/GuestLayout";
import type { SharedProps } from "../../types";
export default function NotFound({ status = 404 }: { status?: number }) {
  const { auth } = usePage<SharedProps>().props;
  const Layout = auth ? AppLayout : GuestLayout;
  return (
    <Layout>
      <Head title={String(status)} />
      <div className="py-16 text-center">
        <p className="eyebrow mb-4">{status}</p>
        <h1 className="page-title">
          {status === 404 ? "This page is missing." : "Something went wrong."}
        </h1>
        <p className="muted my-5">
          {status === 404
            ? "The note may have been removed, or this page is not available to your account."
            : "Please try again in a moment."}
        </p>
        <Link href={auth ? "/notes" : "/login"} className="button">
          {auth ? "Back to notes" : "Back to login"}
        </Link>
      </div>
    </Layout>
  );
}
