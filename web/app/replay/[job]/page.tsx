'use client'
import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'next/navigation'
import AppShell from '../../../components/AppShell'
import LoadingState from '../../../components/LoadingState'
import { api } from '../../../lib/api'

export default function ReplayPage() {
  const params = useParams<{ job: string }>()
  const [job, setJob] = useState<any>(null)
  const [mode, setMode] = useState<'as_scored' | 'recomputed'>('recomputed')
  const [actionState, setActionState] = useState('')
  useEffect(() => {
    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | undefined
    if (params?.job) {
      api.replay(params.job, mode).then((next) => {
        if (cancelled) return
        setJob(next)
        if (next?.partial) {
          timer = setTimeout(() => {
            api.replay(params.job, mode).then((updated) => {
              if (!cancelled) setJob(updated)
            }).catch(() => null)
          }, 1500)
        }
      }).catch(() => null)
    }
    return () => {
      cancelled = true
      if (timer) clearTimeout(timer)
    }
  }, [mode, params?.job])
  const parity = useMemo(() => {
    const result = job?.result || {}
    const avgDelta = Number(result.avg_score_delta ?? 0)
    const highDelta = Number(result.high_count_delta ?? 0)
    return {
      avgDelta,
      highDelta,
      selected: job?.selectedStats || {},
      asScored: result.as_scored || {},
      recomputed: result.recomputed || {},
      status: String(result.parity_status || 'UNKNOWN'),
      explanation: String(result.parity_explanation || 'No replay comparison recorded yet.'),
      matchedCount: Number(result.matched_count || 0),
      mismatchedCount: Number(result.mismatched_count || 0),
      maxScoreDelta: Number(result.max_score_delta || 0),
      commonCauses: Array.isArray(result.common_mismatch_causes) ? result.common_mismatch_causes : [],
      determinismStatus: String(result.determinism_status || 'UNKNOWN'),
      determinismExplanation: String(result.determinism_explanation || 'No prior replay fingerprint comparison recorded yet.'),
      previousReplayId: String(result.previous_replay_id || ''),
    }
  }, [job])
  async function exportBrief() {
    if (!params?.job) return
    setActionState('Exporting replay brief…')
    try {
      const md = await api.replayBrief(params.job, 'md')
      const blob = new Blob([md], { type: 'text/markdown;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `replay-${params.job}-brief.md`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      setActionState(`Replay brief exported: ${params.job}`)
    } catch {
      setActionState('Replay brief export failed')
    }
  }
  return <AppShell title={`Replay ${params?.job || ''}`} subtitle="Replay workspace" subtitleKey="subtitleReplay">
    {!job ? <LoadingState /> : <>
      <div className="grid-3">
        <div className="card"><h3>Status</h3><p>{job.status}</p></div>
        <div className="card"><h3>Provenance</h3><p>{job.modelVersion} / {job.featureSetVersion}</p><p className="muted">{job.replayMode} · {job.watermarkPolicy}</p></div>
        <div className="card"><h3>Parity</h3><p>{parity.status}</p><p className="muted">avg score delta {parity.avgDelta.toFixed(4)} · high-count delta {parity.highDelta}</p><p className="muted">{parity.explanation}</p></div>
      </div>
      <div className="card">
        <p className="muted">metadata-first mode available when recompute artifacts are absent.</p>
        <button onClick={() => setMode('as_scored')}>As Scored</button> <button onClick={() => setMode('recomputed')}>Recomputed</button>
        <button onClick={exportBrief}>Export brief</button>
        <p>Mode: {mode}</p>
        <div className="kv"><span>Selected count</span><strong>{Number(parity.selected.count || 0)}</strong></div>
        <div className="kv"><span>Selected avg score</span><strong>{Number(parity.selected.avg_score || 0).toFixed(4)}</strong></div>
        <div className="kv"><span>Selected high-or-higher</span><strong>{Number(parity.selected.high_or_higher || 0)}</strong></div>
        <div className="kv"><span>As-scored vs recomputed</span><strong>{Number(parity.asScored.count || 0)} / {Number(parity.recomputed.count || 0)}</strong></div>
        <div className="kv"><span>Matched vs mismatched</span><strong>{parity.matchedCount} / {parity.mismatchedCount}</strong></div>
        <div className="kv"><span>Max score delta</span><strong>{parity.maxScoreDelta.toFixed(4)}</strong></div>
        <div className="kv"><span>Determinism</span><strong>{parity.determinismStatus}</strong></div>
        <p className="muted">{parity.determinismExplanation}</p>
        {parity.previousReplayId ? <p className="muted">Compared to prior replay {parity.previousReplayId}.</p> : null}
        {parity.commonCauses.length ? <p className="muted">Common mismatch causes: {parity.commonCauses.join(', ')}</p> : null}
        {job.partial && <p className="muted">Partial replay; limitations: {(job.limitations || []).join(', ')}</p>}
      </div>
      <div className="card"><h3>Deterministic timeline</h3><p>Synchronized summary lanes.</p><p className="muted">Allowed lateness {job.allowedLatenessMs}ms</p></div>
      {actionState ? <div className="card"><p>{actionState}</p></div> : null}
    </>}
  </AppShell>
}
