import { createInertiaApp } from '@inertiajs/inertia-react';
import { InertiaProgress } from '@inertiajs/progress';
import React from 'react';
import ReactDOM from 'react-dom';

InertiaProgress.init();

// see https://vitejs.dev/guide/features.html#glob-import
const pages = import.meta.globEager('./pages/**/*.tsx');

createInertiaApp({
  id: 'app',
  resolve: (name) => pages[`./pages/${name}.tsx`],
  setup({ el, App, props }) {
    ReactDOM.render(
      <App {...props} />,
      el,
    );
  },
}).catch((err) => console.log(err));
