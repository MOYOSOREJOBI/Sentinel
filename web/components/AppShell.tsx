'use client'

import Link from 'next/link'
import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import { useEffect, useMemo, useState } from 'react'
import { api, GlobalFilters, Role } from '../lib/api'
import FilterCombobox from './FilterCombobox'
import { fromDateTimeLocalValue, mergeFilterState, parseFilterState, toDateTimeLocalValue } from '../lib/filterState'
import { requireAuth } from '../lib/requireAuth'
import { allLocales, applyLocale, localeLabel, useI18n } from '../lib/i18n'

const PRIMARY_NAV_ITEMS = [
  { href: '/command-center', label: 'Command Center', key: 'commandCenter' },
  { href: '/queue', label: 'Queue', key: 'queue' },
  { href: '/case', label: 'Cases', key: 'cases' },
  { href: '/trust', label: 'Trust', key: 'trust' },
  { href: '/executive', label: 'Executive', key: 'executive' },
  { href: '/governance', label: 'Governance', key: 'governance', minRole: 'admin' as Role },
  { href: '/feed', label: 'Risk Feed', key: 'feed' },
]

const BOTTOM_NAV_ITEMS = [
  { href: '/about', label: 'About', key: 'about' },
  { href: '/glossary', label: 'Glossary', key: 'glossary' },
]

const FILTER_BAR_ROUTES = new Set([
  '/command-center',
  '/queue',
  '/case',
  '/trust',
  '/executive',
  '/governance',
  '/feed',
])

export function shouldShowFilterBar(pathname?: string | null) {
  return FILTER_BAR_ROUTES.has(String(pathname || ''))
}

export function useGlobalFilters() {
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const filters = useMemo(() => parseFilterState(searchParams), [searchParams])
  const apply = (next: Partial<GlobalFilters>) => {
    const value = mergeFilterState(filters, next)
    const params = new URLSearchParams()
    Object.entries(value).forEach(([k, v]) => {
      if (v !== undefined && v !== null && String(v).trim() !== '') {
        params.set(k, String(v))
      }
    })
    const query = params.toString()
    router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false })
  }
  return { filters, apply }
}

function roleAllowed(userRole: Role, minRole?: Role): boolean {
  if (!minRole) return true
  if (userRole === 'admin') return true
  if (userRole === 'analyst') return minRole !== 'admin'
  return minRole === 'viewer'
}

export default function AppShell({
  title,
  subtitle,
  titleKey,
  subtitleKey,
  children,
  filters,
  setFilters,
}: {
  title: string
  subtitle?: string
  titleKey?: string
  subtitleKey?: string
  children: React.ReactNode
  filters?: GlobalFilters
  setFilters?: (f: Partial<GlobalFilters>) => void
}) {
  const pathname = usePathname()
  const router = useRouter()
  const [me, setMe] = useState<{ email: string; role: Role }>({ email: '', role: 'viewer' })
  const { locale, setLocale, tr } = useI18n()

  useEffect(() => {
    applyLocale(locale)
    requireAuth(router)
      .then((m) => {
        if (!m) return
        setMe({ email: m.email || m.user || '', role: m.role || 'viewer' })
      })
      .catch(() => router.replace('/login'))
  }, [router, locale])

  const logout = async () => {
    await api.logout()
    router.replace('/login')
  }

  const roleLabel = useMemo(() => me.role.charAt(0).toUpperCase() + me.role.slice(1), [me.role])
  const primaryItems = useMemo(() => PRIMARY_NAV_ITEMS.filter((i) => roleAllowed(me.role, i.minRole)), [me.role])
  const bottomItems = useMemo(() => BOTTOM_NAV_ITEMS, [])
  const showFilterBar = Boolean(setFilters && shouldShowFilterBar(pathname))

  const applySuggestion = (field: string, value: string, suggestion?: { kind: string; value: string }) => {
    if (!setFilters) {
      return
    }
    if (!suggestion) {
      if (field === 'symbol') {
        setFilters({ q: value, symbol: '' })
        return
      }
      setFilters({ [field]: value } as Partial<GlobalFilters>)
      return
    }
    switch (suggestion.kind) {
      case 'symbol':
        setFilters({ symbol: suggestion.value, q: '' })
        return
      case 'country':
        setFilters({ countryCode: suggestion.value, q: '' })
        return
      case 'region':
        setFilters({ region: suggestion.value, q: '' })
        return
      case 'sector':
        setFilters({ sector: suggestion.value, q: '' })
        return
      case 'industry':
        setFilters({ industry: suggestion.value, q: '' })
        return
      case 'venue':
        setFilters({ venue: suggestion.value, q: '' })
        return
      default:
        setFilters({ [field]: value } as Partial<GlobalFilters>)
    }
  }

  return (
    <main className="shell-root">
      <aside className="left-nav" data-testid="sidebar">
        <div className="brand"><span className="brand-dot" />Sentinel</div>
        <p className="muted">{tr('operatorWorkbench', 'Operator Workbench')}</p>
        <div className="nav-section-label">{tr('navPrimary', 'Primary')}</div>
        <nav className="left-nav-list">
          {primaryItems.map((item) => (
            <Link key={item.href} href={item.href} className={`left-nav-link ${pathname === item.href ? 'active' : ''}`}>{tr(item.key, item.label)}</Link>
          ))}
        </nav>
        <div className="left-nav-bottom">
          <div className="nav-section-label">{tr('navReference', 'Reference')}</div>
          <nav className="left-nav-list compact">
            {bottomItems.map((item) => (
              <Link key={item.href} href={item.href} className={`left-nav-link ${pathname === item.href ? 'active' : ''}`}>{tr(item.key, item.label)}</Link>
            ))}
          </nav>
          <div className="locale-panel">
            <div className="nav-section-label">{tr('navLanguage', 'Language')}</div>
            <select
              aria-label={tr('localeLabel', 'Locale')}
              data-testid="locale-switcher"
              value={locale}
              onChange={(e)=>{ setLocale(e.target.value as any) }}
            >
              {allLocales().map((l)=><option key={l} value={l}>{localeLabel(l)}</option>)}
            </select>
          </div>
        </div>
      </aside>
      <section className="workspace">
        <header className="topbar sticky-top">
          <div>
            <h1>{titleKey ? tr(titleKey, title) : title}</h1>
            {subtitle ? <p className="muted">{subtitleKey ? tr(subtitleKey, subtitle) : subtitle}</p> : null}
          </div>
          <div className="user-meta">
            <span className="pill role">{roleLabel}</span>
            <span className="pill">{me.email || tr('signedIn', 'signed in')}</span>
            <button data-testid="logout" className="ghost-btn" onClick={logout}>{tr('logout', 'Log out')}</button>
          </div>
        </header>
        {showFilterBar ? (
          <div className="filter-strip sticky-filters">
            <select data-testid="filter-window" value={filters?.window || '24h'} onChange={(e) => setFilters({ window: e.target.value as any })}>
              <option value="1h">{tr('window1h', '1 hour')}</option>
              <option value="24h">{tr('window24h', '24 hours')}</option>
              <option value="7d">{tr('window7d', '7 days')}</option>
              <option value="30d">{tr('window30d', '30 days')}</option>
              <option value="90d">{tr('window90d', '90 days')}</option>
              <option value="1y">{tr('window1y', '1 year')}</option>
              <option value="5y">{tr('window5y', '5 years')}</option>
              <option value="20y">{tr('window20y', '20 years')}</option>
              <option value="custom">{tr('windowCustom', 'custom range')}</option>
            </select>
            {filters?.window === 'custom' ? (
              <>
                <input type="datetime-local" value={toDateTimeLocalValue(filters?.start)} onChange={(e) => setFilters({ start: fromDateTimeLocalValue(e.target.value) })} />
                <input type="datetime-local" value={toDateTimeLocalValue(filters?.end)} onChange={(e) => setFilters({ end: fromDateTimeLocalValue(e.target.value) })} />
              </>
            ) : null}
            <FilterCombobox field="symbol" allFields testId="filter-q" value={filters?.symbol || filters?.q || ''} placeholder={tr('filterPlaceholderSymbol', 'topic / symbol')} onCommit={(value, suggestion) => applySuggestion('symbol', value, suggestion)} />
            <FilterCombobox field="country" testId="filter-country" value={filters?.countryCode || ''} placeholder={tr('filterPlaceholderCountry', 'country')} onCommit={(value) => setFilters({ countryCode: value })} />
            <FilterCombobox field="region" testId="filter-region" value={filters?.region || ''} placeholder={tr('filterPlaceholderRegion', 'region')} onCommit={(value) => setFilters({ region: value })} />
            <FilterCombobox field="sector" testId="filter-sector" value={filters?.sector || ''} placeholder={tr('filterPlaceholderSector', 'sector')} onCommit={(value) => setFilters({ sector: value })} />
            <FilterCombobox field="industry" testId="filter-industry" value={filters?.industry || ''} placeholder={tr('filterPlaceholderIndustry', 'industry')} onCommit={(value) => setFilters({ industry: value })} />
            <FilterCombobox field="venue" testId="filter-venue" value={filters?.venue || ''} placeholder={tr('filterPlaceholderVenue', 'venue')} onCommit={(value) => setFilters({ venue: value })} />
            <button data-testid="filter-clear" className="ghost-btn" onClick={() => { setFilters({ window: '24h', start: '', end: '', symbol: '', countryCode: '', region: '', sector: '', industry: '', venue: '', q: '' }) }}>{tr('filterClear', 'Clear')}</button>
          </div>
        ) : null}
        <div className="page-content">{children}</div>
      </section>
    </main>
  )
}
