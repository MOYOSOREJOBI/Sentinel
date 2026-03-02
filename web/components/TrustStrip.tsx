import Link from 'next/link'

export default function TrustStrip({ trustState='stable', modelUnavailable=false, dqWarning='' }: { trustState?: string; modelUnavailable?: boolean; dqWarning?: string }) {
  const degraded = trustState !== 'stable' || modelUnavailable
  return (
    <section className={`trust-strip ${degraded ? 'degraded' : ''} ${modelUnavailable ? 'unavailable' : ''}`}>
      <div>
        <div className={`badge ${degraded ? 'degraded' : 'stable'}`}>{degraded ? 'Degraded trust' : 'Trust stable'}</div>
        <p>{modelUnavailable ? 'Scoring is running in deterministic fallback mode.' : 'Model-backed scoring available where artifacts are loaded.'}</p>
        <p className="muted">DQ: {dqWarning || 'No active data quality degradations'}</p>
      </div>
      {degraded && <Link href="/trust" className="pill">Open Trust</Link>}
    </section>
  )
}
