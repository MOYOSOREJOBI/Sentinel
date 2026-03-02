'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import AppShell, { useGlobalFilters } from '../../components/AppShell'
import LoadingState from '../../components/LoadingState'
import { api } from '../../lib/api'

type PolicyDraft = {
  id?: string
  name: string
  severity_band: string
  notify_channel: string
  sla_minutes: number
  on_call_target: string
}

const emptyPolicy: PolicyDraft = {
  name: '',
  severity_band: 'elevated',
  notify_channel: 'webhook_stub',
  sla_minutes: 60,
  on_call_target: '',
}

export default function GovernancePage() {
  const [snapshot, setSnapshot] = useState<any>(null)
  const [me, setMe] = useState<any>({ role: 'viewer' })
  const [currentOnCall, setCurrentOnCall] = useState<any>(null)
  const [deploy, setDeploy] = useState({ model_name: 'escalation', version: 'baseline-v1' })
  const [policy, setPolicy] = useState<PolicyDraft>(emptyPolicy)
  const [actionState, setActionState] = useState('')
  const { filters, apply } = useGlobalFilters()

  const isAdmin = me?.role === 'admin'
  const canStartReplay = me?.role === 'admin' || me?.role === 'analyst'

  async function load() {
    const [nextSnapshot, nextMe, nextOnCall] = await Promise.all([
      api.governanceSnapshot(),
      api.me().catch(() => ({ role: 'viewer' })),
      api.currentOnCall().catch(() => null),
    ])
    setSnapshot(nextSnapshot)
    setMe(nextMe)
    setCurrentOnCall(nextOnCall)
  }

  useEffect(() => {
    load().catch(() => null)
  }, [])

  async function runDeploy() {
    setActionState('Deploying model…')
    try {
      await api.deployModel(deploy.model_name, deploy.version)
      await load()
      setActionState(`Model deployed: ${deploy.model_name} ${deploy.version}`)
    } catch {
      setActionState('Model deploy failed or forbidden')
    }
  }

  async function runReplay() {
    setActionState('Starting replay…')
    try {
      const created = await api.startReplay({})
      await load()
      setActionState(`Replay queued: ${created?.id || 'unknown'}`)
    } catch {
      setActionState('Replay start failed or forbidden')
    }
  }

  async function savePolicy() {
    setActionState(policy.id ? 'Updating escalation policy…' : 'Creating escalation policy…')
    try {
      await api.saveEscalationPolicy(policy)
      setPolicy(emptyPolicy)
      await load()
      setActionState('Escalation policy saved')
    } catch {
      setActionState('Escalation policy save failed or forbidden')
    }
  }

  if (!snapshot) {
    return (
      <AppShell title="Governance" subtitle="Admin control plane" titleKey="titleGovernance" subtitleKey="subtitleGovernance" filters={filters} setFilters={apply}>
        <LoadingState />
      </AppShell>
    )
  }

  return (
    <AppShell title="Governance" subtitle="Admin control plane" titleKey="titleGovernance" subtitleKey="subtitleGovernance" filters={filters} setFilters={apply}>
      <div className="grid-3" data-testid="governance-console">
        <section className="card">
          <h3>Model deployments</h3>
          <p className="muted">Active lineage, deployment history, and artifact versions.</p>
          {Array.isArray(snapshot.modelDeployments) && snapshot.modelDeployments.length ? snapshot.modelDeployments.slice(0, 8).map((row: any) => (
            <div key={`${row.id}-${row.model_version}`} className="kv" style={{ alignItems: 'flex-start' }}>
              <span>{row.model_name} {row.model_version}</span>
              <span className="muted">{row.status} · {row.deployed_at || row.created_at || 'pending'}</span>
            </div>
          )) : <div className="empty-state">No model deployments recorded yet.</div>}
          <div style={{ marginTop: 12 }}>
            <h4 style={{ marginBottom: 8 }}>Deploy model</h4>
            {isAdmin ? (
              <div className="form-grid">
                <input value={deploy.model_name} onChange={(e) => setDeploy((d) => ({ ...d, model_name: e.target.value }))} placeholder="model name" />
                <input value={deploy.version} onChange={(e) => setDeploy((d) => ({ ...d, version: e.target.value }))} placeholder="version" />
                <button onClick={runDeploy}>Deploy</button>
              </div>
            ) : <div className="empty-state">Admin role required to deploy models.</div>}
          </div>
        </section>

        <section className="card">
          <h3>Replay jobs</h3>
          <p className="muted">Queue, inspect, and compare replay runs.</p>
          {Array.isArray(snapshot.replayJobs) && snapshot.replayJobs.length ? snapshot.replayJobs.slice(0, 10).map((row: any) => (
            <div key={row.id} className="kv" style={{ alignItems: 'flex-start' }}>
              <span><Link href={`/replay/${row.id}`}>{row.id}</Link></span>
              <span className="muted">{row.status} · {row.replayMode || 'recompute'}</span>
            </div>
          )) : <div className="empty-state">No replay jobs yet.</div>}
          <div style={{ marginTop: 12 }}>
            <h4 style={{ marginBottom: 8 }}>Replay control</h4>
            {canStartReplay ? <button onClick={runReplay}>Start replay</button> : <div className="empty-state">Analyst or admin role required to start replay.</div>}
          </div>
        </section>

        <section className="card">
          <h3>On-call + escalation</h3>
          <p className="muted">Current responder and severity routing policy.</p>
          <div className="kv">
            <span>Current on-call</span>
            <span>{currentOnCall?.currentOnCall || 'unassigned'}</span>
          </div>
          <div className="kv">
            <span>Rotation</span>
            <span className="muted">{currentOnCall?.name || 'No active rotation'}</span>
          </div>
          <div style={{ marginTop: 12 }}>
            <h4 style={{ marginBottom: 8 }}>Editable escalation policies</h4>
            {Array.isArray(snapshot.escalationPolicies) && snapshot.escalationPolicies.length ? snapshot.escalationPolicies.map((row: any) => (
              <div key={row.id} className="queue-row" style={{ marginBottom: 8 }}>
                <div>
                  <strong>{row.name}</strong> · <span className="severity-chip">{row.severityBand}</span>
                  <div className="muted">{row.notifyChannel || 'n/a'} · SLA {row.slaMinutes}m · {row.onCallTarget || 'unassigned'}</div>
                </div>
                {isAdmin ? <button onClick={() => setPolicy({
                  id: row.id,
                  name: row.name,
                  severity_band: row.severityBand,
                  notify_channel: row.notifyChannel || 'webhook_stub',
                  sla_minutes: Number(row.slaMinutes || 60),
                  on_call_target: row.onCallTarget || '',
                })}>Edit</button> : null}
              </div>
            )) : <div className="empty-state">No escalation policies configured yet.</div>}
            {isAdmin ? (
              <div className="form-grid" style={{ marginTop: 8 }}>
                <input value={policy.name} onChange={(e) => setPolicy((p) => ({ ...p, name: e.target.value }))} placeholder="policy name" />
                <select value={policy.severity_band} onChange={(e) => setPolicy((p) => ({ ...p, severity_band: e.target.value }))}>
                  <option value="elevated">elevated</option>
                  <option value="high">high</option>
                  <option value="critical">critical</option>
                </select>
                <input value={policy.notify_channel} onChange={(e) => setPolicy((p) => ({ ...p, notify_channel: e.target.value }))} placeholder="notify channel" />
                <input value={String(policy.sla_minutes)} onChange={(e) => setPolicy((p) => ({ ...p, sla_minutes: Number(e.target.value || 60) }))} placeholder="SLA minutes" />
                <input value={policy.on_call_target} onChange={(e) => setPolicy((p) => ({ ...p, on_call_target: e.target.value }))} placeholder="on-call target" />
                <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                  <button onClick={savePolicy}>{policy.id ? 'Save policy' : 'Create policy'}</button>
                  {policy.id ? <button onClick={() => setPolicy(emptyPolicy)}>Clear selection</button> : null}
                </div>
              </div>
            ) : <div className="empty-state">Admin role required to edit escalation policies.</div>}
          </div>
        </section>
      </div>

      <section className="card" style={{ marginTop: 12 }}>
        <h3>Model registry</h3>
        {Array.isArray(snapshot.modelLineage) && snapshot.modelLineage.length ? snapshot.modelLineage.slice(0, 12).map((row: any) => (
          <div key={`${row.id || row.model}-${row.version}`} className="kv">
            <span>{row.model_name || row.model}</span>
            <span className="muted">{row.version} ({row.state})</span>
          </div>
        )) : <div className="empty-state">No model registry rows available.</div>}
      </section>

      {actionState ? <div className="card" style={{ marginTop: 12 }}><p>{actionState}</p></div> : null}
    </AppShell>
  )
}
