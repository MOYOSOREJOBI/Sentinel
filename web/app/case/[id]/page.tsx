'use client'

import { useEffect, useState } from 'react'
import { api } from '../../../lib/api'
import AppShell, { useGlobalFilters } from '../../../components/AppShell'

export default function CasePage({ params }: { params: { id: string } }) {
  const [row, setRow] = useState<any>(null)
  const [summary, setSummary] = useState<any>(null)
  const [me, setMe] = useState<any>({ role: 'viewer' })
  const [note, setNote] = useState('')
  const [status, setStatus] = useState('investigating')
  const [disposition, setDisposition] = useState('')
  const { filters, apply } = useGlobalFilters()

  const load = () => { api.caseDetail(params.id).then(setRow); api.case(params.id).then(setSummary) }
  useEffect(() => {
    load()
    api.me().then(setMe).catch(() => null)
  }, [params.id])
  const readOnly = me?.role === 'viewer'

  return (
    <AppShell title={`Case ${params.id}`} subtitle="Governed investigation surface with evidence, notes, and disposition history." subtitleKey="subtitleCaseDetail" filters={filters} setFilters={apply}>
      {!row?.id && !summary?.id ? <div className="empty-state">Case unavailable or not found.</div> : <>
        <div className="grid-3">
          <div className="card"><h3>Status</h3><p>{row?.status || summary?.status || 'investigating'}</p></div>
          <div className="card"><h3>Owner</h3><p>{row?.owner || summary?.owner || 'Unassigned'}</p></div>
          <div className="card"><h3>SLA</h3><p>{summary?.updated_at ? `Updated ${summary.updated_at}` : 'Awaiting analyst note'}</p></div>
        </div>
        <div className="grid-2">
          <section className="card" data-testid="case-notes-panel">
            <h3>Evidence and notes</h3>
            <p className="muted">Linked incident: {summary?.incident_id || row?.incident_id || '-'}</p>
            {(row?.evidence || []).length ? <ul>{row.evidence.map((e: any) => <li key={e.id}>{e.evidence_type} · {e.reference_id}</li>)}</ul> : <div className="empty-state">No evidence yet. Attach incident evidence to support disposition.</div>}
            {(row?.notes || []).length ? <ul>{row.notes.map((n: any) => <li key={n.id}>{n.actor}: {n.note}</li>)}</ul> : <div className="empty-state">Awaiting analyst note.</div>}
          </section>
          <section className="card" data-testid="case-audit-trail">
            <h3>Audit trail and timeline</h3>
            {readOnly ? <div className="empty-state">Viewer role is read-only for case workflow updates.</div> : <div className="form-grid" data-testid="case-actions">
              <input value={note} onChange={(e) => setNote(e.target.value)} placeholder="add note" />
              <button onClick={async () => { await api.caseNote(params.id, note).catch(() => null); setNote(''); load() }}>Add note</button>
              <button onClick={async () => { await api.caseEvidence(params.id, 'incident', String(row?.incident_id || summary?.incident_id || ''), { source: 'ui' }).catch(() => null); load() }}>Attach incident evidence</button>
              <select value={status} onChange={(e) => setStatus(e.target.value)}><option>open</option><option>investigating</option><option>escalated</option><option>closed</option></select>
              <button onClick={async () => { await api.caseStatus(params.id, status).catch(() => null); load() }}>Update status</button>
              <input value={disposition} onChange={(e) => setDisposition(e.target.value)} placeholder="disposition reason" />
              <button onClick={async () => { await api.caseDisposition(params.id, 'closed', disposition || 'resolved by analyst').catch(() => null); setDisposition(''); load() }}>Capture disposition</button>
            </div>}
            {(row?.actions || []).length ? <ul>{row.actions.map((a: any) => <li key={a.id}>{a.created_at}: {a.actor} · {a.action}</li>)}</ul> : <div className="empty-state" style={{ marginTop: 10 }}>No activity timeline entries yet.</div>}
          </section>
        </div>
      </>}
    </AppShell>
  )
}
