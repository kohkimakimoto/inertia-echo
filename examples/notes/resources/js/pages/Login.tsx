import { Form, Head } from "@inertiajs/react";
import GuestLayout from "../layouts/GuestLayout";
export default function Login({
  errors = {},
}: {
  errors?: Record<string, string>;
}) {
  return (
    <GuestLayout>
      <Head title="Log in" />
      <h1 className="page-title mb-2">Welcome back</h1>
      <p className="muted mb-8">Log in to your personal notebook.</p>
      <Form action="/login" method="post">
        {({ processing }) => (
          <div className="space-y-5">
            <div>
              <label htmlFor="email">Email address</label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="username"
                defaultValue="demo@example.com"
                required
              />
              {errors.email && <p className="error">{errors.email}</p>}
            </div>
            <div>
              <label htmlFor="password">Password</label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
              />
              {errors.password && <p className="error">{errors.password}</p>}
            </div>
            <button disabled={processing} className="button w-full">
              {processing ? "Logging in…" : "Log in"}
            </button>
          </div>
        )}
      </Form>
      <div className="mt-7 border-t border-stone-200 pt-5 text-xs leading-6 text-stone-500">
        Demo accounts: demo@example.com or alice@example.com
        <br />
        Password: <code>password</code>
      </div>
    </GuestLayout>
  );
}
