# Security Curl Summary

- Proxy login sets both auth and CSRF cookies: see [proxy_login_admin.txt](/Users/mac/Desktop/Sentinel/docs/progress_check/20260304T002604Z/EVIDENCE/security/proxy_login_admin.txt) and [admin.cookies](/Users/mac/Desktop/Sentinel/docs/progress_check/20260304T002604Z/EVIDENCE/security/admin.cookies).
- Proxy-authenticated identity read succeeds (`200 OK`): see [proxy_me_admin.txt](/Users/mac/Desktop/Sentinel/docs/progress_check/20260304T002604Z/EVIDENCE/security/proxy_me_admin.txt).
- Authenticated admin write without `X-CSRF-Token` is rejected (`403 Forbidden`): see [query_backfill_without_csrf.txt](/Users/mac/Desktop/Sentinel/docs/progress_check/20260304T002604Z/EVIDENCE/security/query_backfill_without_csrf.txt).
- Unauthenticated logout is rejected (`401 Unauthorized`): see [logout_without_auth.txt](/Users/mac/Desktop/Sentinel/docs/progress_check/20260304T002604Z/EVIDENCE/security/logout_without_auth.txt).
- Authenticated logout with a valid CSRF token still succeeds (`204 No Content`): see [logout_with_auth_and_csrf.txt](/Users/mac/Desktop/Sentinel/docs/progress_check/20260304T002604Z/EVIDENCE/security/logout_with_auth_and_csrf.txt).
- Viewer stays blocked on admin-only query write even with a valid CSRF token (`403 Forbidden`): see [query_backfill_viewer_forbidden.txt](/Users/mac/Desktop/Sentinel/docs/progress_check/20260304T002604Z/EVIDENCE/security/query_backfill_viewer_forbidden.txt).
