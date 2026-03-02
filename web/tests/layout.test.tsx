import test from 'node:test'
import assert from 'node:assert/strict'
import React from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import LayoutShell from '../app/layout-shell'

test('layout renders html shell and child content', () => {
  const html = renderToStaticMarkup(
    React.createElement(LayoutShell, { children: React.createElement('main', null, 'child content') }),
  )
  assert.match(html, /<html lang=\"en\">/)
  assert.match(html, /child content/)
})
