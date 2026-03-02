'use client'
import AppShell from '../../components/AppShell'
import Link from 'next/link'

const TERMS = [
  ['Incident', 'A clustered risk event that groups multiple alerts into one triage object.'],
  ['Alert', 'A single scored signal that crossed a threshold.'],
  ['Severity', 'Operator-facing band (stable → elevated → high → critical).'],
  ['Escalation probability', 'Model estimate in [0,1] that risk will intensify.'],
  ['Anomaly', 'Deviation from expected short-window behavior.'],
  ['Trust', 'How reliable the current data and model inputs are.'],
  ['Replay', 'Re-run scoring over a historical window for audit and review.'],
  ['Case', 'Investigation workspace linked to an incident.'],
  ['Fallback mode', 'Deterministic scoring path used when model artifacts are unavailable.'],
  ['Model registry', 'Source of model versions, training time, and artifact lineage metadata.'],
  ['Priority score', 'Queue ordering signal in [0,100] balancing urgency and confidence.'],
  ['Composite risk', 'Risk intensity score derived from anomaly + escalation + penalties.'],
  ['Expected loss proxy', 'Queue-level proxy computed from priority and escalation probability.'],
  ['Tail risk (p95)', '95th percentile of composite risk in the selected window.'],
  ['Signal-to-noise', 'Mean risk divided by dispersion (higher means concentrated risk).'],
  ['Confidence-weighted risk', 'Composite risk discounted by confidence.'],
  ['Replay parity', 'Claim that replayed scoring matches live scoring semantics.'],
  ['DQ warning', 'Data-quality degradation signal (missingness, duplicates, late events).'],
]

export default function GlossaryPage() {
  return (
    <AppShell
      title="Glossary"
      subtitle="Plain-English terms for operators and non-experts."
      titleKey="titleGlossary"
      subtitleKey="subtitleGlossary"
    >
      <section className="card">
        <div className="glossary-grid">
          {TERMS.map(([term, definition]) => (
            <div className="glossary-row" key={term}>
              <strong>{term}</strong>
              <span>{definition}</span>
            </div>
          ))}
        </div>
      </section>
      <section className="card">
        <h3>Where to read more</h3>
        <p>See <Link href="/about" className="pill">About</Link> for quantitative formulas, operator playbook, and pipeline context.</p>
      </section>
    </AppShell>
  )
}
