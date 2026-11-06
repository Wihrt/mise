# Tests

## ingress — Exposition HTTP du chart
Branche : `feat/ingress` · Commande : `mise run test:helm`

### ING-002 — Ingress exposé en TLS sur l'hôte configuré
- **Critère** : `docs/specs/2026-10-28-ingress-design.md` › ING-002
- **Test** : `chart/tests/acceptance/ingress_test.yaml` › `[ING-002] tls on configured host`
