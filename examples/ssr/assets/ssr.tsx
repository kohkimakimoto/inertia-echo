import { createInertiaApp, type ResolvedComponent } from '@inertiajs/react'

const pages = import.meta.glob<{ default: ResolvedComponent }>('./pages/**/*.tsx', { eager: true })

createInertiaApp({
  resolve: name => pages[`./pages/${name}.tsx`],
})
