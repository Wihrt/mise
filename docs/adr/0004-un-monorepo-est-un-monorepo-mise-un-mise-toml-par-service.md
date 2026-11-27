# 4. Un monorepo est un monorepo mise, un `mise.toml` par service

Date: 2026-11-27

## Status

Proposed

## Context

- Dans un monorepo, le code vit dans des sous-dossiers, un par service. `repo-stack` ne regarde que la racine et répond `stack=none` : pas de skill de pile, pas de test d'architecture, pas de porte de couverture.
- mise 2026.11.10 gère les monorepos nativement : `monorepo_root = true` et `[monorepo] config_roots` à la racine, un `mise.toml` par service, `mise run '//...:check'` lance chaque service dans son dossier avec ses outils. La découverte automatique est dépréciée (vérifié le 2026-11-27, https://mise.jdx.dev/tasks/monorepo.html).
- La porte de couverture juge déjà un service quand on lui passe son dossier en `--repo`. Lancée à la racine, elle écarte en silence les fichiers Python d'un service (essai du 2026-11-27).

Options :
1. Monorepo natif de mise, un `mise.toml` par service, `config_roots` comme liste des services. Compromis : une liste à tenir, des outils déclarés par service.
2. Un seul `mise.toml` racine, des tâches suffixées par service. Compromis : chaque fragment renommé à la main par service, et les chemins Python restent faux.
3. Détection des services par balayage des manifestes. Compromis : un manifeste qui ne sert qu'aux tests (chart) devient un service ; mise exige la liste de toute façon.

## Decision

Option 1. `config_roots` est la seule liste des services, pour mise comme pour le harnais. `harness local stack` la lit et nomme la pile de chaque service ; chaque service est outillé comme un dépôt simple ; la porte de couverture tourne par service, et refuse un rapport dont aucun fichier n'est suivi.

## Consequences

- **Gagné :** les fragments et les skills de pile s'appliquent tels quels dans chaque service ; la détection reste déterministe et passe en Go, avec des tests ; le faux vert de la porte à la racine disparaît.
- **Coûté :** une ligne par service dans `config_roots`, un `mise.toml` et une tâche `check` par service ; `repo-stack` dépend du binaire `harness`.
- **À revoir si :** mise change ou retire son mode monorepo ; un besoin réel de code à la racine à côté des services, ou de bibliothèques partagées (`go.work`, workspaces uv), apparaît.
