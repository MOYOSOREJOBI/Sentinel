'use client'

import { useEffect, useMemo, useState } from 'react'
import AppShell, { useGlobalFilters } from '../../components/AppShell'
import LoadingState from '../../components/LoadingState'
import { api } from '../../lib/api'
import { subscribeTrustPatches } from '../../lib/stream'

function pct(v: number) {
  return `${(v * 100).toFixed(2)}%`
}

function level(value: number, threshold: number) {
  return value >= threshold ? 'warning' : 'stable'
}

export default function TrustPage() {
  const { filters, apply } = useGlobalFilters()
  const [data, setData] = useState<any>(null)
  const [actionState, setActionState] = useState('')

  useEffect(() => { api.trust(filters).then(setData) }, [JSON.stringify(filters)])
  useEffect(() => subscribeTrustPatches((p) => setData((d: any) => ({ ...(d || {}), ...p }))), [])

  const fallbackMode = Boolean(data?.modelState?.fallback_mode)
  const dq = data?.dqBreakdown || {}
  const calibration = data?.calibration || {}
  const calibrationRows = Array.isArray(calibration.history) && calibration.history.length ? calibration.history : [{
    trained_at: data?.modelState?.last_trained_at || 'pending',
    auc: calibration.auc || 0,
    brier: calibration.brier || 0,
    pr_auc: calibration.pr_auc || 0,
    anomaly_stability: calibration.anomaly_stability || 0,
  }]

  const dqWarnings = useMemo(() => {
    const warnings: string[] = []
    if ((dq.missingness?.value || 0) >= (dq.missingness?.threshold || 1)) warnings.push('Missingness is above the trust threshold.')
    if ((dq.duplicates?.value || 0) >= (dq.duplicates?.threshold || 1)) warnings.push('Duplicate events are high enough to distort rankings.')
    if ((dq.lateEvents?.value || 0) >= (dq.lateEvents?.threshold || 1)) warnings.push('Late events are arriving outside the preferred scoring window.')
    return warnings
  }, [dq])

  async function rerunReplay() {
    setActionState('Queueing replay…')
    try {
      const out = await api.startReplay({})
      setActionState(`Replay queued: ${out?.id || 'unknown'}`)
    } catch {
      setActionState('Replay is unavailable for this role.')
    }
  }

  async function exportBriefHint() {
    const text = 'Use Incident detail to export a markdown incident brief for handoff.'
    await navigator.clipboard.writeText(text).catch(() => {})
    setActionState('Brief export instructions copied')
  }

  return (
    <AppShell title="Trust" subtitle="Model and data quality posture" titleKey="titleTrust" subtitleKey="subtitleTrust" filters={filters} setFilters={apply}>
      {!data ? <LoadingState /> : <>
        <section className="card" data-testid="trust-model-mode">
          <h3>Model mode</h3>
          <p className="muted">This tells you whether Sentinel is using trained model artifacts or a deterministic fallback estimate.</p>
          <div className="grid-3">
            <div className="card kpi-card"><h3>Mode</h3><p>{fallbackMode ? 'Fallback' : 'Trained'}</p><span className="muted">{fallbackMode ? 'Deterministic estimate path' : 'Artifact-backed scoring path'}</span></div>
            <div className="card kpi-card"><h3>Model version</h3><p>{data?.modelState?.model_version || 'unknown'}</p><span className="muted">Lineage for the latest visible score</span></div>
            <div className="card kpi-card"><h3>Trained at</h3><p>{data?.modelState?.last_trained_at || 'unknown'}</p><span className="muted">Last known training timestamp</span></div>
          </div>
        </section>

        <section className="card" data-testid="trust-dq">
          <h3>Data quality</h3>
          <p className="muted">These inputs affect how much confidence you should place in the queue and feed ordering.</p>
          <div className="grid-3">
            <div className="card">
              <h3>Missingness</h3>
              <p>{pct(Number(dq.missingness?.value || 0))}</p>
              <span className="muted">Threshold {pct(Number(dq.missingness?.threshold || 0.02))} · {level(Number(dq.missingness?.value || 0), Number(dq.missingness?.threshold || 0.02))}</span>
            </div>
            <div className="card">
              <h3>Duplicates</h3>
              <p>{pct(Number(dq.duplicates?.value || 0))}</p>
              <span className="muted">Threshold {pct(Number(dq.duplicates?.threshold || 0.01))} · {level(Number(dq.duplicates?.value || 0), Number(dq.duplicates?.threshold || 0.01))}</span>
            </div>
            <div className="card">
              <h3>Late events</h3>
              <p>{pct(Number(dq.lateEvents?.value || 0))}</p>
              <span className="muted">Threshold {pct(Number(dq.lateEvents?.threshold || 0.005))} · {level(Number(dq.lateEvents?.value || 0), Number(dq.lateEvents?.threshold || 0.005))}</span>
            </div>
          </div>
        </section>

        <section className="card" data-testid="trust-calibration">
          <h3>Calibration</h3>
          <p className="muted">Calibration tells you whether the risk ranking behaves like a useful probability signal instead of a raw score only.</p>
          <div className="grid-2">
            <div>
              <div className="kv"><span>AUC</span><span>{Number(calibration.auc || 0).toFixed(3)} <span className="muted">Higher is better separation.</span></span></div>
              <div className="kv"><span>Brier</span><span>{Number(calibration.brier || 0).toFixed(3)} <span className="muted">Lower is better probability accuracy.</span></span></div>
              <div className="kv"><span>PR-AUC</span><span>{Number(calibration.pr_auc || 0).toFixed(3)} <span className="muted">Precision vs recall for rarer escalations.</span></span></div>
              <div className="kv"><span>Stability</span><span>{Number(calibration.anomaly_stability || 0).toFixed(3)} <span className="muted">How consistent anomaly behavior is over time.</span></span></div>
            </div>
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
                <thead>
                  <tr className="muted" style={{ textAlign: 'left' }}>
                    <th style={{ paddingBottom: 6 }}>trained_at</th>
                    <th style={{ paddingBottom: 6 }}>auc</th>
                    <th style={{ paddingBottom: 6 }}>brier</th>
                    <th style={{ paddingBottom: 6 }}>pr_auc</th>
                    <th style={{ paddingBottom: 6 }}>stability</th>
                  </tr>
                </thead>
                <tbody>
                  {calibrationRows.map((row: any, idx: number) => (
                    <tr key={`${row.trained_at}-${idx}`}>
                      <td style={{ padding: '6px 0' }}>{row.trained_at}</td>
                      <td style={{ padding: '6px 0' }}>{Number(row.auc || 0).toFixed(3)}</td>
                      <td style={{ padding: '6px 0' }}>{Number(row.brier || 0).toFixed(3)}</td>
                      <td style={{ padding: '6px 0' }}>{Number(row.pr_auc || 0).toFixed(3)}</td>
                      <td style={{ padding: '6px 0' }}>{Number(row.anomaly_stability || 0).toFixed(3)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </section>

        <section className="card">
          <h3>Why this matters</h3>
          <ul>
            <li>{fallbackMode ? 'Fallback mode is active, so rankings are estimates instead of the trained model path.' : 'Trained model mode is active, so rankings include the current calibrated model bundle.'}</li>
            <li>{dqWarnings.length ? 'High data-quality warnings reduce ranking confidence for this window.' : 'Current data-quality signals are within the normal trust thresholds.'}</li>
          </ul>
        </section>

        <section className="card">
          <h3>What you can do</h3>
          <div className="form-grid">
            <button onClick={() => apply({ ...filters, q: '', window: '1h' })}>Tighten filters</button>
            <button onClick={() => apply({ ...filters, window: '1h' })}>Switch to a shorter window</button>
            <button onClick={rerunReplay}>Rerun replay</button>
            <button onClick={exportBriefHint}>Export brief</button>
          </div>
          <p className="muted" style={{ marginTop: 10 }}>
            Tight filters and shorter windows reduce noise. Replay helps validate whether the current score path still behaves as expected.
          </p>
        </section>

        {actionState ? <div className="card"><p>{actionState}</p></div> : null}
      </>}
    </AppShell>
  )
}
