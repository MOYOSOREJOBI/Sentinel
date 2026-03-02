'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import AppShell, { useGlobalFilters } from '../../components/AppShell'
import { api } from '../../lib/api'

export default function CaseWorkspacePage() {
  const [cases, setCases] = useState<any[]>([])
  const [me, setMe] = useState<any>({ role: 'viewer' })
  const { filters, apply } = useGlobalFilters()

  useEffect(() => {
    api.me().then(setMe).catch(() => null)
    api.cases().then(setCases)
  }, [])

  const readOnly = me?.role === 'viewer'

  return (
    <AppShell title="Case Workspace" subtitle="Investigation workflow with owner, status, evidence, notes, and disposition." titleKey="titleCaseWorkspace" subtitleKey="subtitleCaseWorkspace" filters={filters} setFilters={apply}>
      <div className="card" data-testid="cases-list">
        <h3>Open investigations</h3>
        {!cases.length ? <div className="empty-state">No cases yet. Promote an incident to start investigation.</div> : cases.map((c) => (
          <div className="queue-row" key={c.id}>
            <div><strong>Case #{c.id}</strong> · incident {c.incident_id} · <span className="severity-chip">{c.status}</span></div>
            <div className="muted">{c.reason}</div>
            <div><Link className="pill" href={`/case/${c.id}`}>Open workspace</Link></div>
          </div>
        ))}
      </div>
      <div className="card">
        <h3>Role controls</h3>
        <p>{readOnly ? 'Viewer mode: read-only case access.' : 'Analyst/Admin mode: workflow mutation controls enabled in case detail.'}</p>
      </div>
    </AppShell>
  )
}
