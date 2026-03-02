'use client'

import { useEffect, useMemo, useRef, useState } from 'react'
import { api } from '../lib/api'

type Suggestion = { kind: string; value: string; label: string }

export default function FilterCombobox({
  field,
  value,
  placeholder,
  allFields = false,
  testId,
  onCommit,
}: {
  field: string
  value: string
  placeholder: string
  allFields?: boolean
  testId?: string
  onCommit: (value: string, suggestion?: Suggestion) => void
}) {
  const [query, setQuery] = useState(value)
  const [open, setOpen] = useState(false)
  const [activeIndex, setActiveIndex] = useState(0)
  const [items, setItems] = useState<Suggestion[]>([])
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setQuery(value)
  }, [value])

  useEffect(() => {
    const term = query.trim()
    if (term.length < 2) {
      setItems([])
      setOpen(false)
      setActiveIndex(0)
      return
    }
    let cancelled = false
    const timer = setTimeout(() => {
      api.searchSuggest(term, 10, allFields ? '' : field).then((res: any) => {
        if (cancelled) {
          return
        }
        const next = Array.isArray(res?.items) ? res.items : []
        setItems(next)
        setOpen(next.length > 0)
        setActiveIndex(0)
      }).catch(() => {
        if (cancelled) {
          return
        }
        setItems([])
        setOpen(false)
      })
    }, 150)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [allFields, field, query])

  useEffect(() => {
    const handle = (evt: MouseEvent) => {
      if (!rootRef.current?.contains(evt.target as Node)) {
        setOpen(false)
      }
    }
    window.addEventListener('mousedown', handle)
    return () => window.removeEventListener('mousedown', handle)
  }, [])

  const visibleItems = useMemo(() => {
    if (allFields) {
      return items
    }
    return items.filter((item) => item.kind === field)
  }, [allFields, field, items])

  const commit = (nextValue: string, suggestion?: Suggestion) => {
    setQuery(nextValue)
    setOpen(false)
    onCommit(nextValue, suggestion)
  }

  return (
    <div ref={rootRef} className="filter-combobox">
      <input
        data-testid={testId}
        value={query}
        placeholder={placeholder}
        onFocus={() => setOpen(visibleItems.length > 0)}
        onChange={(e) => {
          setQuery(e.target.value)
          setOpen(true)
        }}
        onBlur={() => {
          window.setTimeout(() => {
            commit(query)
          }, 0)
        }}
        onKeyDown={(e) => {
          if (!visibleItems.length) {
            if (e.key === 'Enter') {
              e.preventDefault()
              commit(query)
            }
            return
          }
          if (e.key === 'ArrowDown') {
            e.preventDefault()
            setOpen(true)
            setActiveIndex((idx) => (idx + 1) % visibleItems.length)
          } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            setOpen(true)
            setActiveIndex((idx) => (idx - 1 + visibleItems.length) % visibleItems.length)
          } else if (e.key === 'Enter') {
            e.preventDefault()
            commit(visibleItems[activeIndex]?.value || query, visibleItems[activeIndex])
          } else if (e.key === 'Escape') {
            setOpen(false)
          }
        }}
      />
      {open && visibleItems.length > 0 ? (
        <div className="combobox-menu" role="listbox">
          {visibleItems.map((item, idx) => (
            <button
              key={`${item.kind}-${item.value}`}
              type="button"
              className={`combobox-option ${idx === activeIndex ? 'active' : ''}`}
              onMouseEnter={() => setActiveIndex(idx)}
              onMouseDown={(e) => {
                e.preventDefault()
                commit(item.value, item)
              }}
            >
              <span>{item.value}</span>
              <span className="muted">{item.kind}</span>
            </button>
          ))}
        </div>
      ) : null}
    </div>
  )
}
