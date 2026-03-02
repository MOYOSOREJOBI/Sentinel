'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '../lib/api'
import LoginScreen from '../components/LoginScreen'

export default function HomePage() {
  const router = useRouter()

  useEffect(() => {
    api.me().then(() => router.replace('/command-center')).catch(() => null)
  }, [router])

  return <LoginScreen />
}
