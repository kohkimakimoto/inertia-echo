/**
 * Thin Inertia SSR entry for in-process runtimes (slowjs).
 *
 * Unlike the Node path (@inertiajs/react/server createServer), this file
 * only exposes renderPage(page) -> { head, body }. No HTTP server.
 */
import { createElement, type ComponentType } from 'react'
import { renderToString } from 'react-dom/server'
import { App } from '@inertiajs/react'
import { buildSSRBody, type Page } from '@inertiajs/core'

type PageModule = { default: ComponentType<any> }

const pages = import.meta.glob<{ default: ComponentType<any> }>('./pages/**/*.tsx', { eager: true }) as Record<string, PageModule>

function resolveComponent(name: string): ComponentType<any> {
  const mod = pages[`./pages/${name}.tsx`]
  if (!mod) {
    throw new Error(`Inertia SSR: page not found: ${name}`)
  }
  return mod.default || (mod as unknown as ComponentType<any>)
}

function renderPage(page: Page): { head: string[]; body: string } {
  let head: string[] = []
  const initialComponent = resolveComponent(page.component)
  const reactApp = createElement(App, {
    initialPage: page,
    initialComponent,
    resolveComponent: (name: string) => resolveComponent(name),
    onHeadUpdate: (elements: string[]) => {
      head = elements
    },
  })
  const html = renderToString(reactApp)
  return {
    head,
    body: buildSSRBody('app', page, html),
  }
}

;(globalThis as typeof globalThis & { renderPage: typeof renderPage }).renderPage = renderPage
