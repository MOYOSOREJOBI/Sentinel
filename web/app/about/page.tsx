'use client'
import AppShell from '../../components/AppShell'

const WHAT_SENTINEL_DOES = [
  'Watches streaming telemetry (price-like ticks + derived features).',
  'Scores risk per symbol using anomaly + escalation models.',
  'Converts scores into alerts, then groups alerts into incidents.',
  'Ranks incidents into a queue and a Twitter-like Risk Feed.',
  'Supports cases for investigation, replay for audit, and governance controls.',
]

const RISK_MEANS = [
  'Anomaly: how far current behavior deviates from recent history.',
  'Escalation probability: likelihood the situation will intensify.',
  'Confidence: how reliable the score inputs are right now.',
  'Freshness: newer events matter more.',
  'Trust penalty: data-quality issues reduce how much we believe the score.',
]

const PIPELINE = [
  '1) Ticks: raw events arrive (think “telemetry per symbol”).',
  '2) Candles: ticks are summarized into time buckets (open, high, low, close, volume).',
  '3) Features: candles become model inputs (returns, z-scores, volatility, drift signals).',
  '4) Scoring: models output anomaly, escalation, and composite risk.',
  '5) Alerts: score spikes create alerts with reason codes.',
  '6) Incidents: related alerts are grouped into an incident you can triage.',
  '7) Cases: incidents can be promoted into an investigation workspace.',
  '8) Replay + audit: you can rerun the scoring path and verify audit integrity.',
]

const REALTIME_VS_HISTORICAL = [
  'Real-time: new ticks, new scores, and feed/queue updates (via SSE, with polling fallback).',
  'Historical: candles, scores, incidents, and cases stored in Postgres/Timescale for replay and analysis.',
]

const QUICKSTART = [
  '1) Pick a time window and a symbol/topic.',
  '2) Watch the Risk Feed for what is moving now.',
  '3) Use Queue to prioritize by expected loss proxy and escalation probability.',
  '4) Promote to a Case when signals stay elevated and evidence is consistent.',
  '5) Use Trust to confirm the data and model mode before acting.',
]

export default function AboutPage() {
  return (
    <AppShell
      title="About Sentinel"
      subtitle="Real-time risk signals, incident workflow, and an explainable feed for operators and everyday users."
      titleKey="titleAbout"
      subtitleKey="subtitleAbout"
    >
      <section className="card">
        <h3>What Sentinel does</h3>
        <ul className="copy-list">
          {WHAT_SENTINEL_DOES.map((line) => <li key={line}>{line}</li>)}
        </ul>
      </section>

      <section className="card">
        <h3>What “risk” means here</h3>
        <p>Risk is a ranked signal that something is unusual AND important.</p>
        <p>Sentinel combines:</p>
        <ul className="copy-list">
          {RISK_MEANS.map((line) => <li key={line}>{line}</li>)}
        </ul>
      </section>

      <section className="card">
        <h3>The pipeline (plain English, but technically correct)</h3>
        <ul className="copy-list">
          {PIPELINE.map((line) => <li key={line}>{line}</li>)}
        </ul>
      </section>

      <section className="card">
        <h3>What is real-time vs what is historical</h3>
        <ul className="copy-list">
          {REALTIME_VS_HISTORICAL.map((line) => <li key={line}>{line}</li>)}
        </ul>
      </section>

      <section className="card">
        <h3>Honest scope</h3>
        <p>This build is production-minded but still evolving.</p>
        <p>Some governance actions are intentionally non-destructive.</p>
        <p>Where models are unavailable, Sentinel can run in fallback mode and will label it.</p>
      </section>

      <section className="card">
        <h3>Operator quickstart</h3>
        <ul className="copy-list">
          {QUICKSTART.map((line) => <li key={line}>{line}</li>)}
        </ul>
      </section>
    </AppShell>
  )
}
