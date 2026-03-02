import './globals.css'

export const metadata = {
  title: 'Sentinel Dashboard',
  description: 'Real-time anomaly detection and monitoring platform',
}

import LayoutShell from './layout-shell'

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <LayoutShell>{children}</LayoutShell>
}
