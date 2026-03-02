'use client'
import { useEffect, useMemo, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import AppShell from '../../../components/AppShell'
import WhyThisFiredPanel from '../../../components/WhyThisFiredPanel'
import { api } from '../../../lib/api'

export default function IncidentPage() {
  const params = useParams<{ id: string }>()
  const router = useRouter()
  const [data, setData] = useState<any>(null)
  const [brief, setBrief] = useState<any>(null)
  const [actionState, setActionState] = useState('')

  useEffect(() => {
    if (!params?.id) return
    api.incident(params.id).then(setData)
    api.incidentBrief(params.id).then(setBrief).catch(() => setBrief(null))
  }, [params?.id])

  const timeline = useMemo(() => {
    const events = [
      { label: 'Created', value: data?.created_at || data?.started_at || 'Unknown' },
      { label: 'Last activity', value: data?.updated_at || data?.last_activity_at || 'Unknown' },
      { label: 'Status', value: data?.status || 'Unknown' },
      { label: 'Severity', value: data?.severity || 'Unknown' },
    ]
    return events
  }, [data])

  async function runAction(kind: 'ack' | 'resolve' | 'escalate' | 'notify') {
    if (!params?.id) return
    setActionState(`${kind}…`)
    try {
      if (kind === 'notify') {
        await api.notifyOnCall(params.id)
      } else {
        await api.incidentTransition(params.id, kind)
      }
      const next = await api.incident(params.id)
      setData(next)
      setActionState(`${kind} complete`)
    } catch {
      setActionState(`${kind} failed or forbidden`)
    }
  }

  async function promote() {
    if (!params?.id) return
    setActionState('Promoting…')
    try {
      const created = await api.promoteIncidentCase(params.id, 'promoted from incident detail')
      if (created?.case_id) {
        setActionState(`Promoted as case #${created.case_id}`)
        router.push(`/case/${created.case_id}`)
        return
      }
      setActionState('Promoted')
    } catch {
      setActionState('Promotion failed or forbidden')
    }
  }

  async function copyBrief() {
    if (!params?.id) return
    const md = await api.incidentBrief(params.id, 'md').catch(() => '')
    if (!md) return
    await navigator.clipboard.writeText(md).catch(() => {})
    setActionState('Brief copied')
  }

  async function downloadBrief() {
    if (!params?.id) return
    const md = await api.incidentBrief(params.id, 'md').catch(() => '')
    if (!md) return
    const blob = new Blob([md], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `incident-${params.id}-brief.md`
    a.click()
    URL.revokeObjectURL(url)
    setActionState('Brief downloaded')
  }

  return (
    <AppShell title={`Incident ${params?.id || ''}`} subtitle="Score evidence, operations, and comms" subtitleKey="subtitleIncidentDetail">
      <section className="card" data-testid="incident-ops">
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: '1rem', flexWrap: 'wrap' }}>
          <div>
            <h3 style={{ margin: 0 }}>Incident Ops</h3>
            <p className="muted" style={{ margin: '0.25rem 0 0' }}>Status {data?.status || '-'} · Severity {data?.severity || '-'}</p>
          </div>
          <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
            <button onClick={() => runAction('ack')}>Ack</button>
            <button onClick={() => runAction('resolve')}>Resolve</button>
            <button onClick={() => runAction('escalate')}>Escalate</button>
            <button onClick={() => runAction('notify')}>Notify on-call</button>
            <button onClick={promote} data-testid="incident-promote-case">Promote to Case</button>
          </div>
        </div>
        {actionState ? <p className="muted">{actionState}</p> : null}
      </section>

      <section className="card">
        <h3>State timeline</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(12rem,1fr))', gap: '0.75rem' }}>
          {timeline.map((item) => (
            <div key={item.label} style={{ border: '1px solid var(--border, #334)', borderRadius: '0.5rem', padding: '0.75rem' }}>
              <div className="muted" style={{ fontSize: '0.75rem' }}>{item.label}</div>
              <div style={{ fontWeight: 600 }}>{String(item.value)}</div>
            </div>
          ))}
        </div>
      </section>

      <div className="card"><h3>Score header</h3><p>Priority {data?.score_header?.priority ?? '-'}</p></div>
      <section className="card"><h3>Explanation</h3><p>{brief?.whatHappened || data?.explanation_text || '-'}</p></section>
      <section className="card"><h3>Model lineage</h3><p>{data?.model_version || 'unknown'}</p></section>

      <section className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: '1rem', flexWrap: 'wrap' }}>
          <div>
            <h3 style={{ margin: 0 }}>Incident brief</h3>
            <p className="muted" style={{ margin: '0.25rem 0 0' }}>Exportable operator handoff for comms.</p>
          </div>
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            <button onClick={copyBrief}>Copy brief</button>
            <button onClick={downloadBrief}>Download markdown</button>
          </div>
        </div>
        {brief ? (
          <div style={{ marginTop: '0.75rem' }}>
            <p>{brief.whatHappened}</p>
            <p className="muted">Drivers: {(brief.topDrivers || []).join(' · ') || 'n/a'}</p>
            <p className="muted">Recommended action: {brief.recommendedAction || 'watch'}</p>
          </div>
        ) : <p className="muted">Brief unavailable.</p>}
      </section>

      <WhyThisFiredPanel drivers={brief?.topDrivers || data?.top_drivers || []} caveats={data?.caveats || []} />
    </AppShell>
  )
}
