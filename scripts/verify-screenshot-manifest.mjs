#!/usr/bin/env node
import fs from 'node:fs/promises'

const manifestPath = process.argv[2] || 'docs/screenshots/manifest.json'
const requiredRoutes = ['/command-center','/queue','/incident','/case','/case/','/trust','/replay/','/governance','/executive','/about','/glossary']

const raw = await fs.readFile(manifestPath, 'utf8').catch(() => '')
if (!raw) {
  console.error(`FAIL: missing manifest at ${manifestPath}`)
  process.exit(1)
}
let manifest
try { manifest = JSON.parse(raw) } catch (e) {
  console.error(`FAIL: invalid manifest JSON (${e.message})`)
  process.exit(1)
}
if (!Array.isArray(manifest)) {
  console.error('FAIL: manifest must be an array')
  process.exit(1)
}

const failures = manifest.filter(x => x?.status === 'FAIL')
if (failures.length) {
  console.error(`FAIL: manifest contains failed routes (${failures.length})`)
  process.exit(1)
}

for (const routePrefix of requiredRoutes) {
  const row = manifest.find(x => typeof x?.route === 'string' && x.route.startsWith(routePrefix))
  if (!row) {
    console.error(`FAIL: required route missing in manifest (${routePrefix})`)
    process.exit(1)
  }
  if (row.status === 'PASS') {
    if (!row.output) {
      console.error(`FAIL: PASS route missing output path (${routePrefix})`)
      process.exit(1)
    }
  } else if (row.status !== 'SKIP') {
    console.error(`FAIL: unexpected route status for ${routePrefix}: ${row.status}`)
    process.exit(1)
  }
}

console.log('PASS: screenshot manifest verified')
