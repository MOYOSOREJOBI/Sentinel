'use client'

import { useI18n } from '../lib/i18n'

export default function RealtimeStatus({
  sseStatus,
  polling,
  label = 'Realtime',
}: {
  sseStatus: 'connecting' | 'open' | 'closed' | 'error'
  polling: boolean
  label?: string
}) {
  const { tr } = useI18n()
  let tone = 'offline'
  let text = tr('statusOffline', 'Offline')

  if (sseStatus === 'open') {
    tone = 'connected'
    text = tr('statusConnected', 'Connected')
  } else if (polling || sseStatus === 'error' || sseStatus === 'connecting') {
    tone = 'degraded'
    text = tr('statusDegraded', 'Degraded')
  }

  return (
    <div className={`status-pill ${tone}`} data-testid="realtime-status">
      <span className="status-dot" />
      {label}: {text}
      {tone === 'degraded' ? ' (polling)' : ''}
    </div>
  )
}
