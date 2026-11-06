# Harness Claude Code dans le dépôt mise — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: `stacked-tdd-mr` (wrapper over superpowers:subagent-driven-development). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** versionner le harness Claude Code dans `Wihrt/mise` et le déployer en liens par l'étape `dotfiles` de `mise bootstrap`.

**Architecture:** `claude/` et `rtk/` à la racine du dépôt ; entrées `[dotfiles]` `mode = "symlink"` dans `config.toml` ; l'outillage *du dépôt* (lint, tests) vit dans `.config/mise.toml`, jamais dans `mise.toml` ni `tasks/`, qui seraient chargés comme config globale.

**Tech Stack:** mise 2026.10.25 (`[dotfiles]`, tâches inline), bash, bats 1.13.0, jq, prek, cocogitto, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-11-06-harness-in-mise-design.md`

## Global Constraints

- Une PR par tâche, empilées (Task N basée sur Task N-1), un seul worktree `~/.config/mise-wt-harness`. Task 0 part de `origin/main` sur la branche existante `feat/harness-import` (porte déjà la spec et ce plan).
- Aucun trailer `Co-Authored-By: Claude`, aucune ligne « Generated with Claude Code ».
- **Fait vérifié (mise 2026.10.25, `MISE_CONFIG_DIR` de test)** : dans le dossier de config globale, `mise.toml` et `mise.local.toml` sont chargés depuis n'importe quel répertoire ; `.config/mise.toml` et `.mise.toml` ne le sont que dans le dépôt. Les fichiers de `tasks/` sont des tâches globales. Donc : outillage du dépôt dans `.config/mise.toml`, tâches inline ; `tasks/` ne reçoit que des tâches destinées au poste (`bootstrap`, `claude-plugins`).
- Tous les liens : `mode = "symlink"`, jamais `symlink-each`.
- Hors dépôt : `~/.claude/docs/`, `projects/`, `plugins/`, `skills/synced/`, `skills/curl-md`, `skills/find-skills`, `skills/.archive/`, `*.bak*`, `~/.config/rtk/llm-proxy/`.
- Scripts bash : `set -euo pipefail`, propres à `shellcheck`.
- Aucune action sur le `$HOME` réel de l'utilisateur ; la migration (spec, « Migration de ce poste ») vient après merge, sur feu vert.

## Review Focus

1. `mise bootstrap` lancé sur ce poste **avant** migration : les vrais fichiers de `~/.claude` bloquent l'étape `dotfiles` (`refusing to overwrite`) ; le README (Task 3) doit dire de migrer d'abord. Vérifié en revue.
2. `HOME` où `~/.claude` n'existe pas : `mise dot apply` crée les parents. Couvert par HAR-001 (HOME vide).
3. Nom de skill avec un point : la cohérence skills ↔ `[dotfiles]` (Task 3) lit `config.toml` en TOML, pas par grep. Test unitaire bats : skill `a.b` sans entrée → échec nommant `a.b`.
4. Hook (Task 2) : `file_path` contenant `..` ou lien pendant vers le main checkout → deny, code de sortie 0. Test unitaire bats.
5. `claude plugin list` qui échoue (non connecté) dans `claude-plugins` (Task 4) : avertissement, code 0. Test unitaire bats.

---

### Task 0: Outillage du dépôt et import du harness — branche `feat/harness-import`

**Mode:** `chore(tooling)` (pas de critère ; gate = `mise run check` + `prek run --all-files`)

**Files:**
- Create: `claude/{CLAUDE.md,RTK.md,settings.json}`, `claude/agents/{acceptance-writer,implementer,mutant-writer}.md`, `claude/hooks/` (`*.sh`, `mise.toml`, `tests/*.bats`, `tests/helpers.bash`), `claude/skills/<17 skills>/`, `rtk/{config,filters,pipeline}.toml`, `rtk/pipelines/`
- Create: `.config/mise.toml`, `.pre-commit-config.yaml`, `.editorconfig`, `cog.toml`, `.gitattributes`, `_typos.toml` (gabarits `~/.claude/skills/repo-tooling/templates/`)
- Modify: `.gitignore` (ajout `.tdd/`), `.github/workflows/ci.yml` (job `check` du skill repo-tooling ; garder le job existant)

**Interfaces:**
- Produces: `mise run test:hooks` (bats `claude/hooks/tests/`, rapport `.tdd/junit.xml`) ; `mise run lint:shell` (shellcheck sur `tasks/*` et `claude/hooks/*.sh`) ; `mise run check`. Les tâches suivantes ajoutent `lint:harness`, `test:dotfiles`, `test:plugins` dans `.config/mise.toml`.

- [ ] **Step 1:** copie depuis le poste : `rsync -a --exclude='*.bak*' --exclude=.archive --exclude=synced`, skills = dossiers réels seulement (pas `curl-md`, `find-skills`). Vérifier : `find claude rtk \( -name '*.bak*' -o -type l \) | wc -l` → `0` ; `ls claude/skills | wc -l` → `17`.
- [ ] **Step 2:** baseline repo-tooling dans `.config/mise.toml` (et non `mise.toml`). Vérifier depuis `/tmp` : `mise tasks ls | grep -c 'test:hooks'` → `0`.
- [ ] **Step 3:** `mise run check` → code 0, 114 tests bats `ok` ; `prek run --all-files` → code 0 (scoper les hooks qui échouent sur les fichiers importés et nommer l'exclusion dans la PR).
- [ ] **Step 4:** commit `chore(tooling): outillage du dépôt et import du harness`, PR vers `main`, CI verte.

### Task 1: Hygiène du harness — branche `feat/harness-hygiene`, slug `harness-hygiene`, préfixe `HAR`

**Criteria:** HAR-006 · **Frontière :** tâche mise `lint:harness`

**Files:**
- Modify: `.config/mise.toml` (tâche `lint:harness`, script `.config/lint-harness.sh` si plus de 3 lignes)

**Interfaces:**
- Produces: `mise run lint:harness` — code 0 si propre, sinon non nul avec une ligne `lint:harness: <fichier>: <raison>` par problème. Task 3 y ajoute la cohérence.

- [ ] **Step 1:** `lint:harness` échoue sur tout `*.bak*` sous `claude/`/`rtk/` et tout fichier contenant `/home/arnaud` hors `claude/skills/*/references/`.
- [ ] **Step 2:** `mise run check` → code 0.
- [ ] **Step 3:** commit, PR (base `feat/harness-import`).

### Task 2: Le hook résout les liens — branche `feat/harness-hook-realpath`, slug `harness-hook-realpath`

**Criteria:** HAR-002 · **Frontière :** entrée JSON / sortie du hook

**Files:**
- Modify: `claude/hooks/require-worktree.sh` (chemin Edit/Write/NotebookEdit)
- Test (unitaire, Review Focus 4): `claude/hooks/tests/require-worktree.bats`

- [ ] **Step 1:** résoudre `file_path` avec `realpath -m` avant la remontée `dirname` et le test `in_main_checkout` ; `git check-ignore` porte sur le chemin résolu.
- [ ] **Step 2:** `mise run test:hooks` → tous `ok`.
- [ ] **Step 3:** commit `fix(hooks): require-worktree résout les liens`, PR.

### Task 3: Déploiement par `[dotfiles]` — branche `feat/harness-dotfiles`, slug `harness-dotfiles`

**Criteria:** HAR-001, HAR-003, HAR-004 · **Frontière :** CLI mise sur un `HOME` temporaire ; tâche `lint:harness`

**Files:**
- Modify: `config.toml` section `[dotfiles]` (ligne 102) : une entrée par fichier, par dossier et par skill (spec, « Déploiement »), commentaire « un skill = une ligne ».
- Modify: `.config/mise.toml` : `lint:harness` gagne la cohérence `claude/skills/*` ↔ `[dotfiles]` et l'existence de chaque `source` ; nouvelle tâche `test:dotfiles` (`mise dot apply --yes` dans un `HOME` temporaire, vérification `realpath` de chaque cible).
- Test (unitaire, Review Focus 3): bats sous `.config/tests/`
- Modify: `README.md` : section « Harness Claude Code » (contenu, ajout d'un skill, migration avant `mise bootstrap`).

**Interfaces:**
- Produces: entrées `[dotfiles]` clé = cible `~/…`, `source` relative à la racine du dépôt.

- [ ] **Step 1:** entrées `[dotfiles]`.
- [ ] **Step 2:** cohérence dans `lint:harness`, lecture TOML structurée.
- [ ] **Step 3:** `test:dotfiles`. Non vérifié : comment faire lire le `config.toml` du dépôt à mise sous `HOME=<tmp>` (`MISE_CONFIG_DIR`, testé en bac à sable pour les tâches ; à confirmer pour `[dotfiles]`). Le vérifier d'abord ; noter la variable retenue dans le rapport.
- [ ] **Step 4:** `mise run check` → code 0 ; le job CI `mise bootstrap --dry-run --yes` reste vert.
- [ ] **Step 5:** README, commit `feat(harness): déployer le harness par mise dotfiles`, PR.

### Task 4: Installation du plugin superpowers — branche `feat/harness-plugin`, slug `harness-plugin`

**Criteria:** HAR-005 · **Frontière :** tâche `claude-plugins`, `claude` simulé

**Files:**
- Create: `tasks/claude-plugins` (`#MISE description="Installe les plugins Claude Code manquants"`), tâche du poste, donc dans `tasks/`
- Modify: `tasks/bootstrap` (dernière étape : `mise run claude-plugins`)
- Modify: `.config/mise.toml` : `test:plugins` (bats `.config/tests/claude-plugins.bats`, Review Focus 5)

**Interfaces:**
- Produces: `mise run claude-plugins`, idempotente, liste de plugins en tête de script : `superpowers@anthropic-plugin-directory`.

- [ ] **Step 1:** pour chaque plugin : sans `claude` sur le `PATH`, rien ; si `claude plugin list` échoue, avertit et continue ; si l'id est absent de la sortie, `claude plugin install <id>`.
- [ ] **Step 2:** `mise run check` → code 0.
- [ ] **Step 3:** commit `feat(bootstrap): installer les plugins Claude Code`, PR.
