'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '../lib/api'
import { applyLocale, localeFromStorage, withLocalePath } from '../lib/i18n'

export default function LoginScreen() {
  const router = useRouter()
  const [email, setEmail] = useState('admin@sentinel.local')
  const [password, setPassword] = useState('Sentinel#123')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const submit = async () => {
    setError('')
    setLoading(true)
    try {
      await api.login(email, password)
      const me = await api.me()
      const preferred = me?.preferredLocale || localeFromStorage()
      applyLocale(preferred)
      router.replace(withLocalePath('/command-center', preferred))
    } catch (e: any) {
      setError(e.message || 'Authentication failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="login-root" data-testid="login-screen">
      <section className="login-card">
        <h1>Sentinel</h1>
        <p>Risk intelligence operator workbench for ranked incidents, governed cases, trust monitoring, and replay metadata.</p>
        <div className="form-grid">
          <input
            data-testid="login-email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="email"
          />
          <input
            data-testid="login-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="password"
          />
          <button data-testid="login-submit" disabled={loading} onClick={submit}>
            {loading ? 'Signing in…' : 'Sign in to Sentinel'}
          </button>
          {error ? <p className="error-text">{error}</p> : null}
        </div>
      </section>
    </main>
  )
}
