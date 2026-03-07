import './globals.css'

export const dynamic = 'force-dynamic'

export const metadata = {
  title: 'Sentinel Dashboard',
  description: 'Real-time anomaly detection and monitoring platform',
}

import LayoutShell from './layout-shell'
import { Suspense } from 'react'

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <Suspense>
      <LayoutShell>{children}</LayoutShell>
    </Suspense>
  )
}
