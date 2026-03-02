#!/usr/bin/env node
import fs from 'node:fs/promises'
import process from 'node:process'
import { chromium } from 'playwright'

const BASE = process.env.BASE_URL || 'http://localhost:3000'
const APIQ = process.env.API_QUERY || `${BASE}/api/proxy/query`
const APIA = process.env.API_ALERTS || `${BASE}/api/proxy/alerts`
const results = []

async function waitReady(url, name, tries = 60) {
  for (let i = 0; i < tries; i++) {
    try { const r = await fetch(url); if (r.ok) return true } catch {}
    await new Promise(r => setTimeout(r, 1000))
  }
  results.push({ flow: name, status: 'FAIL', reason: `timeout waiting ${url}` })
  return false
}

async function login(page, email, password) {
  await page.goto(`${BASE}/`, { waitUntil: 'networkidle' })
  await page.getByPlaceholder('email').fill(email)
  await page.getByPlaceholder('password').fill(password)
  await page.getByRole('button', { name: 'Sign in to Sentinel' }).click()
  await page.waitForLoadState('networkidle')
}

async function resolveIncidentId(request) {
  const req = await request.get(`${APIQ}/queue`, { failOnStatusCode: false })
  if (!req.ok()) return null
  const queue = await req.json()
  return queue[0]?.id || null
}

;(async () => {
  if (!(await waitReady(`${BASE}/`, 'web')) || !(await waitReady(`${APIQ}/readyz`, 'query'))) {
    await fs.mkdir('docs/screenshots', { recursive: true })
    await fs.writeFile('docs/screenshots/browser-validation.json', JSON.stringify(results, null, 2))
    process.exit(1)
  }

  const browser = await chromium.launch({ headless: true })
  const context = await browser.newContext()
  const page = await context.newPage()

  try {
    await login(page, 'admin@sentinel.local', 'Sentinel#123')
  } catch (e) {
    results.push({ flow: 'login-admin', status: 'FAIL', reason: String(e.message || e) })
  }

  // Flow A: world-map filter propagation
  try {
    const seen = []
    page.on('request', req => {
      const u = req.url()
      if (u.includes('/command-center?') || u.includes('/world-map?') || u.includes('/queue?')) seen.push(u)
    })
    await page.goto(`${BASE}/command-center`, { waitUntil: 'networkidle' })
    const first = page.locator('.map-cell').first()
    await first.waitFor({ timeout: 15000 })
    const before = seen.length
    await first.click()
    await page.waitForTimeout(750)
    const propagated = seen.slice(before).find(u => u.includes('region=') || u.includes('countryCode='))
    if (!propagated) throw new Error('no country/region filter observed in downstream fetches')
    results.push({ flow: 'world-map-filter-propagation', status: 'PASS', request: propagated })
  } catch (e) {
    results.push({ flow: 'world-map-filter-propagation', status: 'FAIL', reason: String(e.message || e) })
  }

  // Flow B/C: incident->case + role gating + backend reject
  try {
    const incidentId = await resolveIncidentId(context.request)
    if (!incidentId) throw new Error('no incident id available')

    await page.goto(`${BASE}/incident/${incidentId}`, { waitUntil: 'networkidle' })
    await page.getByRole('button', { name: 'Promote to Case' }).click()
    await page.waitForTimeout(600)

    const msg = await page.locator('section.card >> text=/Case|Promotion blocked/').first().innerText().catch(() => '')
    let caseId = null
    const hit = msg.match(/Case\s+(\d+)/)
    if (hit) caseId = hit[1]
    if (!caseId) {
      const conflict = await context.request.post(`${APIA}/incidents/${incidentId}/promote-case`, { data: { reason: 'duplicate-check' }, failOnStatusCode: false })
      if (conflict.status() === 409) {
        const body = await conflict.json().catch(() => ({}))
        caseId = body?.existing_case_id ? String(body.existing_case_id) : null
      }
    }
    if (!caseId) throw new Error('case id not resolved from promote or duplicate path')

    await page.goto(`${BASE}/case/${caseId}`, { waitUntil: 'networkidle' })
    await page.locator('text=Actions and history').first().waitFor({ timeout: 5000 })

    const viewerCtx = await browser.newContext()
    const viewerPage = await viewerCtx.newPage()
    await login(viewerPage, 'viewer@sentinel.local', 'Sentinel#123')
    await viewerPage.goto(`${BASE}/incident/${incidentId}`, { waitUntil: 'networkidle' })
    const disabled = await viewerPage.locator('text=Viewer mode: case promotion disabled.').count()
    const reject = await viewerCtx.request.post(`${APIA}/incidents/${incidentId}/promote-case`, { data: { reason: 'should-fail' }, failOnStatusCode: false })
    await viewerCtx.close()
    if (disabled < 1 || reject.status() !== 403) throw new Error('viewer UI/API gating failed')

    results.push({ flow: 'incident-to-case-and-role-gating', status: 'PASS', incident_id: incidentId, case_id: caseId })
  } catch (e) {
    results.push({ flow: 'incident-to-case-and-role-gating', status: 'FAIL', reason: String(e.message || e) })
  }

  await browser.close()
  await fs.mkdir('docs/screenshots', { recursive: true })
  await fs.writeFile('docs/screenshots/browser-validation.json', JSON.stringify(results, null, 2))
  for (const r of results) console.log(`${r.status}: ${r.flow}${r.reason ? ` (${r.reason})` : ''}`)
  process.exit(results.some(r => r.status === 'FAIL') ? 1 : 0)
})()
