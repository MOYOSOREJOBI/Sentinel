const fs = require('fs')
const path = require('path')

const req = ['en','fr','es','pt','it','zh-Hans','ja','ar','de','sw','yo','ig','ha','tr','ru','nl','hi']
const dir = path.join(__dirname, '..', 'messages')
const files = fs.readdirSync(dir).filter((f) => f.endsWith('.json')).map((f) => f.replace('.json',''))
const missingLocales = req.filter((l) => !files.includes(l))
if (missingLocales.length) {
  console.error('missing locales:', missingLocales.join(','))
  process.exit(1)
}
const dict = {}
for (const l of req) {
  const fp = path.join(dir, `${l}.json`)
  dict[l] = JSON.parse(fs.readFileSync(fp,'utf8'))
}
const all = new Set()
for (const l of req) Object.keys(dict[l]).forEach((k) => all.add(k))
let bad = 0
for (const l of req) {
  const miss = [...all].filter((k) => !(k in dict[l]))
  if (miss.length) {
    bad++
    console.error(`locale ${l} missing keys: ${miss.join(',')}`)
  }
}
if (bad) process.exit(1)
console.log(`i18n check passed for ${req.length} locales with ${all.size} keys`)
