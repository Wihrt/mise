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

## harness-dotfiles — Déploiement du harness par [dotfiles]
Branche : `feat/harness-dotfiles-2` · Commande : `mise run test:acceptance`

### HAR-001 — Poste vierge : le harness est posé en liens vers le dépôt
- **Critère** : `docs/superpowers/specs/2026-11-06-harness-in-mise-design.md` › HAR-001
- **Test** : `.config/tests/acceptance/harness-dotfiles.bats` › `HAR-001 every harness target is a link resolving into the repo`

### HAR-004 — Un skill sans entrée de déploiement fait échouer le contrôle
- **Critère** : `docs/superpowers/specs/2026-11-06-harness-in-mise-design.md` › HAR-004
- **Test** : `.config/tests/acceptance/harness-dotfiles.bats` › `HAR-004 a skill directory without a [dotfiles] entry fails and is named`

## harness-dotfiles-preserve — Contenus tiers préservés
Branche : `feat/harness-dotfiles-preserve` · Commande : `mise run test:acceptance`

### HAR-003 — Les contenus tiers de `~/.claude` et `~/.config/rtk` restent intacts
- **Critère** : `docs/superpowers/specs/2026-11-06-harness-in-mise-design.md` › HAR-003
- **Test** : `.config/tests/acceptance/harness-dotfiles.bats` › `HAR-003 third-party content in ~/.claude and ~/.config/rtk is left intact`

## harness-plugin — Installation du plugin superpowers
Branche : `feat/harness-plugin-2` · Commande : `mise run test:acceptance`

### HAR-005 — Le plugin superpowers est installé s'il manque, et seulement alors
- **Critère** : `docs/superpowers/specs/2026-11-06-harness-in-mise-design.md` › HAR-005
- **Test** : `.config/tests/acceptance/harness-plugin.bats` › `HAR-005 plugin absent from claude plugin list: installed exactly once`
