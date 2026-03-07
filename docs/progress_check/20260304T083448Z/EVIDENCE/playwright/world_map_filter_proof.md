Playwright proof for the geo/map fix.

- Test: `clicking a country on the map carries the filter into queue`
- Base URL: `http://localhost:33000`
- Result: `PASS`
- Screenshot: `world-map-filtered-queue.png`
- Console log: `playwright_map_filter.txt`

What this proves:

- The Command Center renders the real world map UI.
- Clicking a country chip applies the country filter in the shared filter state.
- Navigating to Queue preserves the selected `countryCode` in the URL.
- The Queue page remains filtered to rows matching that country.
