import { test, expect } from '@playwright/test'

const E2E_BASE_URL = process.env.E2E_BASE_URL || 'http://localhost:3000'

async function loginAs(page: any, email: string, password = 'Sentinel#123') {
  await page.goto('/login')
  await page.getByTestId('login-email').fill(email)
  await page.getByTestId('login-password').fill(password)
  await page.getByTestId('login-submit').click()
  await expect(page).toHaveURL(/command-center/)
}

async function login(page: any) {
  await loginAs(page, 'admin@sentinel.local')
}

async function fetchJson(page: any, path: string) {
  return page.evaluate(async (target) => {
    const res = await fetch(target, { credentials: 'include', cache: 'no-store' })
    const text = await res.text()
    let body: any = null
    if (text) {
      try {
        body = JSON.parse(text)
      } catch {
        body = text
      }
    }
    return {
      status: res.status,
      body,
    }
  }, path)
}

test('login works and logout clears the session', async ({ page }) => {
  await login(page)
  await expect(page.getByTestId('command-center')).toBeVisible()
  await page.getByTestId('logout').click()
  await expect.poll(async () => {
    return page.evaluate(async () => {
      const res = await fetch('/api/proxy/gateway-api/me', { credentials: 'include', cache: 'no-store' })
      return res.status
    })
  }, { timeout: 15000 }).toBe(401)
})

test('command center loads map and globe containers', async ({ page }) => {
  await login(page)
  await expect(page.getByTestId('command-center')).toBeVisible()
  await expect(page.getByTestId('kpi-open-incidents')).toBeVisible()
  await expect(page.getByTestId('kpi-high-risk')).toBeVisible()
  await expect(page.getByTestId('risk-globe')).toBeVisible()
  await expect(page.getByTestId('world-map')).toBeVisible()
  await expect(page.locator('svg[aria-label="world risk map"]')).toBeVisible()
})

test('queue renders ranked rows after login', async ({ page }) => {
  await login(page)
  await page.goto('/queue')
  await expect(page.getByTestId('queue')).toBeVisible()
  await expect(page.locator('[data-testid="queue"] tbody tr').first()).toBeVisible({ timeout: 15000 })
})

test('switch locale to Arabic sets rtl html direction and updates visible UI copy', async ({ page }) => {
  await login(page)
  await expect(page.getByRole('heading', { name: 'Command Center' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Queue' })).toBeVisible()
  await page.getByTestId('locale-switcher').selectOption('ar')
  await expect(page.locator('html')).toHaveAttribute('dir', 'rtl')
  await expect(page.getByRole('heading', { name: 'مركز القيادة' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'الطابور' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'تسجيل الخروج' })).toBeVisible()
  await page.screenshot({ path: 'test-results/i18n-arabic-command-center.png', fullPage: true })
})

test('SSE endpoint returns a live stream response', async ({ page }) => {
  await login(page)
  const stream = await page.evaluate(async () => {
    const res = await fetch('/api/sse/command-center', {
      credentials: 'include',
      headers: { Accept: 'text/event-stream' },
    })
    const contentType = res.headers.get('content-type') || ''
    await res.body?.cancel()
    return { status: res.status, contentType }
  })
  expect(stream.status).toBe(200)
  expect(stream.contentType).toContain('text/event-stream')
})

test('unauthenticated alert ack returns 401', async ({ request }) => {
  const res = await request.post(`${E2E_BASE_URL}/api/proxy/alerts/alerts/1/ack`)
  expect(res.status()).toBe(401)
})

test('viewer cannot ack alerts', async ({ page }) => {
  await loginAs(page, 'viewer@sentinel.local')
  const result = await page.evaluate(async () => {
    const csrf = document.cookie.split(';').map(v => v.trim()).find(v => v.startsWith('sentinel_csrf='))?.split('=')[1] || ''
    const res = await fetch('/api/proxy/alerts/alerts/1/ack', {
      method: 'POST',
      credentials: 'include',
      headers: csrf ? { 'X-CSRF-Token': decodeURIComponent(csrf) } : {},
    })
    return res.status
  })
  expect(result).toBe(403)
})

test('T2: filter typeahead triggers search-suggest request', async ({ page }) => {
  await login(page)
  const [req] = await Promise.all([
    page.waitForRequest(r => r.url().includes('/search-suggest') && r.url().includes('q='), { timeout: 10000 }),
    page.getByTestId('filter-q').fill('BT'),
  ])
  expect(req.url()).toContain('q=')
})

test('T3: changing time window to 7d keeps page functional', async ({ page }) => {
  await login(page)
  await page.getByTestId('filter-window').selectOption('7d')
  // Page stays on command-center, data sections remain
  await expect(page.getByTestId('command-center')).toBeVisible({ timeout: 15000 })
  await expect(page.getByTestId('kpi-open-incidents')).toBeVisible()
})

test('feed window changes send truthful params for 1h and 30d', async ({ page }) => {
  await login(page)
  await page.goto('/feed')
  const [req1] = await Promise.all([
    page.waitForRequest(r => r.url().includes('/api/proxy/query/feed') && r.url().includes('window=1h'), { timeout: 10000 }),
    page.getByTestId('filter-window').selectOption('1h'),
  ])
  expect(req1.url()).toContain('window=1h')

  const [req2] = await Promise.all([
    page.waitForRequest(r => r.url().includes('/api/proxy/query/feed') && r.url().includes('window=30d'), { timeout: 10000 }),
    page.getByTestId('filter-window').selectOption('30d'),
  ])
  expect(req2.url()).toContain('window=30d')
})

test('feed country filter changes returned rows', async ({ page }) => {
  await login(page)
  await page.goto('/feed')
  const initial = await fetchJson(page, '/api/proxy/query/feed?window=24h&limit=50')
  expect(initial.status).toBe(200)
  const body = initial.body
  expect(Array.isArray(body.items)).toBe(true)
  expect(body.items.length).toBeGreaterThan(0)
  const country = 'XX'

  await page.goto(`/feed?window=24h&countryCode=${encodeURIComponent(country)}`)
  await expect(page.getByTestId('filter-country')).toHaveValue(country)
  const filtered = await fetchJson(page, `/api/proxy/query/feed?window=24h&limit=50&countryCode=${encodeURIComponent(country)}`)
  expect(filtered.status).toBe(200)
  const filteredBody = filtered.body
  expect(Array.isArray(filteredBody.items)).toBe(true)
  expect(filteredBody.items.length).toBeGreaterThan(0)
  for (const item of filteredBody.items) {
    expect(String(item.country_code || item.countryCode)).toBe(country)
  }
})

test('feed symbol filter preserves rank reasons', async ({ page }) => {
  await login(page)
  await page.goto('/feed')
  const initial = await fetchJson(page, '/api/proxy/query/feed?window=24h&limit=50')
  expect(initial.status).toBe(200)
  const body = initial.body
  const first = Array.isArray(body.items) ? body.items.find((item: any) => item.symbol) : null
  expect(first).toBeTruthy()
  const symbol = String(first.symbol)

  await page.goto(`/feed?window=24h&q=${encodeURIComponent(symbol)}`)
  await expect(page.getByTestId('filter-q')).toHaveValue(symbol)
  const filtered = await fetchJson(page, `/api/proxy/query/feed?window=24h&limit=50&q=${encodeURIComponent(symbol)}`)
  expect(filtered.status).toBe(200)
  const filteredBody = filtered.body
  expect(Array.isArray(filteredBody.items)).toBe(true)
  expect(filteredBody.items.length).toBeGreaterThan(0)
  for (const item of filteredBody.items) {
    expect(Array.isArray(item.rank_reason || item.rankReason)).toBe(true)
    expect((item.rank_reason || item.rankReason).length).toBeGreaterThan(0)
  }
})

test('T4: candle-risk-panel renders and /scores returns data', async ({ page }) => {
  await login(page)
  await page.goto('/viz')
  const feedResp = await fetchJson(page, '/api/proxy/query/feed?window=24h&limit=10')
  expect(feedResp.status).toBe(200)
  const symbol = Array.isArray(feedResp.body?.items) ? String(feedResp.body.items.find((item: any) => item.symbol)?.symbol || '') : ''
  expect(symbol).not.toBe('')
  const scoresResp = await fetchJson(page, `/api/proxy/query/scores?window=24h&limit=320&symbol=${encodeURIComponent(symbol)}`)
  expect(scoresResp.status).toBe(200)
  const body = scoresResp.body
  expect(Array.isArray(body.series)).toBe(true)
  expect(body.series.length).toBeGreaterThan(0)

  // Panel container must be visible
  await expect(page.getByTestId('candle-risk-panel')).toBeVisible({ timeout: 15000 })
})

test('case workflow creates from incident, adds note, changes status, and shows audit trail', async ({ page }) => {
  await login(page)
  const queueResp = await fetchJson(page, '/api/proxy/query/queue?window=24h')
  expect(queueResp.status).toBe(200)
  const casesResp = await fetchJson(page, '/api/proxy/query/cases')
  expect(casesResp.status).toBe(200)

  const existing = new Set(
    Array.isArray(casesResp.body) ? casesResp.body.map((row: any) => String(row.incident_id)) : [],
  )
  const candidate = Array.isArray(queueResp.body)
    ? queueResp.body.find((row: any) => row?.id && !existing.has(String(row.id)))
    : null
  expect(candidate).toBeTruthy()

  await page.goto(`/incident/${candidate.id}`)
  await page.getByTestId('incident-promote-case').click()
  await expect(page).toHaveURL(/\/case\/\d+/, { timeout: 15000 })

  const note = `playwright note ${Date.now()}`
  await expect(page.getByTestId('case-actions')).toBeVisible()
  await page.getByPlaceholder('add note').fill(note)
  await page.getByRole('button', { name: 'Add note' }).click()
  await expect(page.getByTestId('case-notes-panel')).toContainText(note)

  await page.getByTestId('case-actions').locator('select').selectOption('escalated')
  await page.getByRole('button', { name: 'Update status' }).click()
  await expect(page.getByTestId('case-audit-trail')).toContainText('status')
  await expect(page.getByTestId('case-audit-trail')).toContainText('note')
})
