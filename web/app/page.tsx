'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '../lib/api'
import LoginScreen from '../components/LoginScreen'
import { localeFromStorage, withLocalePath } from '../lib/i18n'

export default function HomePage() {
  const router = useRouter()

  useEffect(() => {
    api.me().then((me) => {
      const preferred = me?.preferredLocale || localeFromStorage()
      router.replace(withLocalePath('/command-center', preferred))
    }).catch(() => null)
  }, [router])

  return <LoginScreen />
}
