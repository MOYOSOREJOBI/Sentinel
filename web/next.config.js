/** @type {import('next').NextConfig} */
module.exports = {
  output: 'standalone',
  reactStrictMode: true,
  poweredByHeader: false,
  eslint: {
    // ESLint 10 is incompatible with Next.js 14 built-in linting.
    // Type checks run separately via tsc --noEmit.
    ignoreDuringBuilds: true,
  },
}
