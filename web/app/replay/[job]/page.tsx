'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import AppShell from '../../../components/AppShell'
import LoadingState from '../../../components/LoadingState'
import { api } from '../../../lib/api'

export default function ReplayPage() {
  const params = useParams<{ job: string }>()
  const [job, setJob] = useState<any>(null)
  const [mode, setMode] = useState<'as_scored' | 'recomputed'>('recomputed')
  useEffect(() => { if (params?.job) api.replay(params.job).then(setJob) }, [params?.job])
  return <AppShell title={`Replay ${params?.job || ''}`} subtitle="Replay workspace" subtitleKey="subtitleReplay">
    {!job ? <LoadingState /> : <>
      <div className="grid-3"><div className="card"><h3>Status</h3><p>{job.status}</p></div><div className="card"><h3>Provenance</h3><p>{job.modelVersion} / {job.featureSetVersion}</p></div><div className="card"><h3>Watermark</h3><p>{job.watermarkPolicy} / {job.allowedLatenessMs}ms</p></div></div>
      <div className="card"><p className="muted">metadata-first mode available when recompute artifacts are absent.</p><button onClick={() => setMode('as_scored')}>As Scored</button> <button onClick={() => setMode('recomputed')}>Recomputed</button><p>Mode: {mode}</p>{job.partial && <p className="muted">Partial replay; limitations: {(job.limitations || []).join(', ')}</p>}</div><div className="card"><h3>Deterministic timeline</h3><p>Synchronized summary lanes.</p></div>
    </>}
  </AppShell>
}
