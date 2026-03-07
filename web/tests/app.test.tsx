import test from 'node:test'
import assert from 'node:assert/strict'
import React from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import fs from 'node:fs'
import TrustStrip from '../components/TrustStrip'
import WorldRiskMap from '../components/WorldRiskMap'
import { shouldShowFilterBar } from '../components/AppShell'

test('command center renders analytics containers', () => {
  const src = fs.readFileSync(process.cwd() + '/app/command-center/page.tsx', 'utf-8')
  assert.match(src, /WorldRiskMap/)
  assert.match(src, /Open incidents/)
  assert.match(src, /Backlog delta/)
})

test('queue renders recommended actions and priority', () => {
  const src = fs.readFileSync(process.cwd() + '/app/queue/page.tsx', 'utf-8')
  assert.match(src, /recommendedAction|recommended_action/)
  assert.match(src, /Priority/)
})

test('incident page renders model and explanation sections and case promotion', () => {
  const src = fs.readFileSync(process.cwd() + '/app/incident/[id]/page.tsx', 'utf-8')
  assert.match(src, /Model lineage/)
  assert.match(src, /Explanation/)
  assert.match(src, /Promote to Case/)
})

test('case workspace and detail show role-gated workflow', () => {
  const workspace = fs.readFileSync(process.cwd() + '/app/case/page.tsx', 'utf-8')
  const detail = fs.readFileSync(process.cwd() + '/app/case/[id]/page.tsx', 'utf-8')
  assert.match(workspace, /Viewer mode: read-only case access/)
  assert.match(detail, /Viewer role is read-only/)
  assert.match(detail, /Capture disposition/)
})

test('trust page renders trend containers', () => {
  const src = fs.readFileSync(process.cwd() + '/app/trust/page.tsx', 'utf-8')
  assert.match(src, /Data quality/)
  assert.match(src, /Why this matters/)
  assert.match(src, /What you can do/)
  assert.match(src, /Artifact hash/)
})

test('replay page labels metadata-first mode and deterministic timeline', () => {
  const src = fs.readFileSync(process.cwd() + '/app/replay/[job]/page.tsx', 'utf-8')
  assert.match(src, /metadata-first mode/)
  assert.match(src, /Deterministic timeline/)
  assert.match(src, /Export brief/)
  assert.match(src, /Matched vs mismatched/)
})

test('governance executive about and glossary pages are productized', () => {
  const gov = fs.readFileSync(process.cwd() + '/app/governance/page.tsx', 'utf-8')
  const exec = fs.readFileSync(process.cwd() + '/app/executive/page.tsx', 'utf-8')
  const about = fs.readFileSync(process.cwd() + '/app/about/page.tsx', 'utf-8')
  const glossary = fs.readFileSync(process.cwd() + '/app/glossary/page.tsx', 'utf-8')
  assert.match(gov, /Editable escalation policies/)
  assert.match(gov, /Replay control/)
  assert.match(gov, /parity/)
  assert.match(exec, /Plain-English/)
  assert.match(about, /Risk is a ranked signal that something is unusual AND important/)
  assert.match(glossary, /DQ warning/)
})

test('filter bar allowlist excludes about and glossary', () => {
  assert.equal(shouldShowFilterBar('/command-center'), true)
  assert.equal(shouldShowFilterBar('/queue'), true)
  assert.equal(shouldShowFilterBar('/about'), false)
  assert.equal(shouldShowFilterBar('/glossary'), false)
})

test('TrustStrip renders fallback honesty', () => {
  const html = renderToStaticMarkup(React.createElement(TrustStrip, { trustState: 'degraded', modelUnavailable: true }))
  assert.match(html, /deterministic fallback mode/)
})

test('WorldRiskMap renders legend and empty state scaffolding', () => {
  const empty = renderToStaticMarkup(React.createElement(WorldRiskMap, { rows: [] }))
  assert.match(empty, /No regional aggregates yet/)
  const html = renderToStaticMarkup(React.createElement(WorldRiskMap, { data: [{ countryIso2: 'US', countryCode: 'US', countryName: 'United States', incidentCount: 3, avgCompositeRisk: 0.62, maxSafetyLevel: 'High Risk', trustState: 'healthy' }] }))
  assert.match(html, /Top regions/)
  assert.match(html, /data-country-code="US"/)
  assert.doesNotMatch(html, /fallback codes/)
})
