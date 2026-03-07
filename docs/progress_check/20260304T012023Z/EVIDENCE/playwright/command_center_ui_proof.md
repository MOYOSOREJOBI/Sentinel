Playwright UI proof captured against `http://localhost:33000/en/command-center` after the final fixes.

Observed state:
- Activity feed rendered `Stream: Connected` and no raw `upstream 500` text.
- The Next dev overlay error toast was no longer visible after the locale-label hydration fix.
- Market + risk overlay rendered live data instead of the empty fallback:
  - `310 score pts`
  - `19 candle pts`
  - `5m resolution`
  - `REIT-010`
  - `quant-v1-2026-03-01`
- Country bar chart rendered 8 real countries: `US`, `AU`, `SG`, `DE`, `GB`, `JP`, `BR`, `SA`.
- Composite histogram rendered non-degenerate bins across `0-20%`, `20-40%`, `40-60%`, `60-80%`, `80-100%`.
- Severity pie chart rendered 4 severities: `critical`, `high`, `elevated`, `stable`.

Browser console state at the end of the verification:
- `0` error messages reported by Playwright console capture.
