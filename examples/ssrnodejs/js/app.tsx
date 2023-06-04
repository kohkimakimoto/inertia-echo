import "../css/app.css";
import { hydrateRoot } from 'react-dom/client'
import { createInertiaApp } from '@inertiajs/react';

createInertiaApp({
  resolve: name => {
    const pages = import.meta.glob('./pages/**/*.tsx', { eager: true })
    return pages[`./pages/${name}.tsx`]
  },
  setup({ el, App, props }) {
    hydrateRoot(el, <App {...props} />)
  },
})
