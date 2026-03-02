# Completion audit note

## Already real
- Core migrations for replay jobs, model deployments, watchlists/preferences already exist and are applied in SQL migration set.
- Query service has end-to-end world map, queue, trust, command center, and executive endpoints with cache-aside wrappers.
- Feature service already computes return, z-score, EWMA, realized vol, and DQ metrics and emits snapshot hashes.

## Partial
- Query filter contract was partially implemented (`time_window`, `country`, `region`, `industry`) but missing required fields (`from/to`, `sector`, `venue`, `symbol`, `locale`).
- Cache keys did not include all filter dimensions, causing cross-filter cache collisions.
- Tests only covered default window parsing.

## Misleading/fake risks
- Filter support in the UI and backend looked broader than what was actually enforced by SQL predicates.

## Breakages noted/fixed in this pass
- Added full filter contract parsing and propagation in query service.
- Added SQL predicate support for sector/venue/symbol and custom time windows.
- Extended cache key dimensions to prevent stale cross-filter responses.
- Added parser test coverage for full contract.

## File edit plan executed
1. Extend `QueueFilters` and centralize SQL `whereClause` construction.
2. Update query loaders and handlers to pass the full filter contract.
3. Update cache key helpers to include all filter dimensions.
4. Update frontend filter type contract to match backend.
5. Add test for full query filter parsing.

## Additional validation + wiring fixes (this pass)
- Verified full unit stack (`make unit`) and found concrete TypeScript contract drift in UI filter wiring (`time_window`/`country` stale fields).
- Fixed UI to use canonical filter keys (`window`, `countryCode`) across Command Center, Queue, Trust, and global AppShell filter strip.
- Re-ran Go/Python/Web unit suites after fixes; all pass.
- Ran integration checks in this environment; scripts correctly reported SKIP when Docker/services were unavailable.
