export default function LoadingState({ label = 'Loading…' }: { label?: string }) {
  return <div className="empty-state">{label}</div>
}
