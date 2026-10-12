import { createInertiaApp } from '@inertiajs/react'

createInertiaApp({
  resolve: name => {
    const pages = import.meta.glob('./pages/**/*.jsx')
    return pages[`./pages/${name}.jsx`]()
  },
})
