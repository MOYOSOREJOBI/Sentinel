export default function WhyThisFiredPanel({ drivers = [], caveats = [] as string[] }: { drivers?: any[]; caveats?: string[] }) {
  return (
    <section className="card">
      <h3>Why this fired</h3>
      <ul>{drivers.map((d: any, i: number) => <li key={i}>{d.feature || d.driver || String(d)}</li>)}</ul>
      {caveats.length > 0 && <p className="muted">{caveats.join('; ')}</p>}
    </section>
  )
}
