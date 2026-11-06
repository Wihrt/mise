# Tests

## harness-hygiene — Hygiène du harness versionné
Branche : `feat/harness-hygiene-3` · Commande : `mise run test:acceptance`

### HAR-006 — Rien de personnel ni d'interne versionné
- **Critère** : `docs/superpowers/specs/2026-11-06-harness-in-mise-design.md` › HAR-006
- **Test** : `.config/tests/acceptance/harness-hygiene.bats` › `HAR-006 a denylisted pattern fails, names the file and hides the pattern`

## harness-hook-realpath — Le hook résout les liens avant de décider
Branche : `feat/harness-hook-realpath` · Commande : `mise run test:acceptance`

### HAR-002 — Une édition via un lien vers le main checkout est refusée
- **Critère** : `docs/superpowers/specs/2026-11-06-harness-in-mise-design.md` › HAR-002
- **Test** : `.config/tests/acceptance/harness-hook-realpath.bats` › `HAR-002 Edit through a file link to the main checkout is denied`
