import { Form, Head } from "@inertiajs/react";
import AppLayout from "../layouts/AppLayout";
import type { SettingsData } from "../types";
export default function Settings({
  settings,
  errors = {},
}: {
  settings: SettingsData;
  errors?: Record<string, string>;
}) {
  return (
    <AppLayout>
      <Head title="Settings" />
      <p className="eyebrow mb-2">Make it yours</p>
      <h1 className="page-title mb-8">Settings</h1>
      <Form
        action="/settings"
        method="post"
        transform={(data) => ({ ...data, page_size: Number(data.page_size) })}
        key={`${settings.theme}-${settings.page_size}`}
      >
        {({ processing }) => (
          <section className="panel max-w-2xl space-y-6">
            <div>
              <label htmlFor="theme">Appearance</label>
              <select name="theme" id="theme" defaultValue={settings.theme}>
                <option value="system">Use system setting</option>
                <option value="light">Light</option>
                <option value="dark">Dark</option>
              </select>
              {errors.theme && <p className="error">{errors.theme}</p>}
            </div>
            <div>
              <label htmlFor="page_size">Notes per page</label>
              <select
                name="page_size"
                id="page_size"
                defaultValue={settings.page_size}
              >
                {[5, 10, 20].map((size) => (
                  <option key={size} value={size}>
                    {size} notes
                  </option>
                ))}
              </select>
              <p className="muted mt-2">
                Controls how many notes load at a time as you scroll.
              </p>
              {errors.page_size && <p className="error">{errors.page_size}</p>}
            </div>
            <button disabled={processing} className="button">
              {processing ? "Saving…" : "Save preferences"}
            </button>
          </section>
        )}
      </Form>
      <section id="body" className="mt-8 max-w-2xl scroll-mt-8">
        <h2 className="text-sm font-semibold">Preferences that follow you</h2>
        <p className="muted mt-2 leading-6">
          These preferences are stored in your account's JSON data. Inertia's
          once prop caches them between visits; saving refreshes the cache. This
          section is also the #body destination for the fragment demonstration.
        </p>
      </section>
    </AppLayout>
  );
}
