# Harness Claude Code dans le dépôt mise — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: `stacked-tdd-mr` (wrapper over superpowers:subagent-driven-development). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** versionner le harness Claude Code dans `Wihrt/mise` et le déployer en liens par l'étape `dotfiles` de `mise bootstrap`.

**Architecture:** `claude/` et `rtk/` à la racine du dépôt ; entrées `[dotfiles]` `mode = "symlink"` dans `config.toml` ; une tâche fichier `tasks/check/harness` (nom `check:harness`) regroupe les contrôles et tourne en CI.

**Tech Stack:** mise 2026.10.25 (`[dotfiles]`, file tasks), bash, bats 1.13.0, jq, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-11-06-harness-in-mise-design.md`

## Global Constraints

- Une PR par tâche, empilées (Task N basée sur Task N-1), un seul worktree `~/.config/mise-wt-harness`.
- Aucun trailer `Co-Authored-By: Claude`, aucune ligne « Generated with Claude Code ».
- Tous les liens : `mode = "symlink"`, jamais `symlink-each`.
- Hors dépôt : `~/.claude/docs/`, `projects/`, `plugins/`, `skills/synced/`, `skills/curl-md`, `skills/find-skills`, `skills/.archive/`, `*.bak*`, `~/.config/rtk/llm-proxy/`.
- Scripts bash : `set -euo pipefail`, propres à `shellcheck` (l'étape CI existante est étendue à tout `tasks/`).
- Aucune action sur le `$HOME` réel de l'utilisateur pendant l'exécution du plan ; la migration (spec, « Migration de ce poste ») vient après merge, sur feu vert.

## Review Focus

1. `mise bootstrap` lancé sur ce poste **avant** migration : les vrais fichiers de `~/.claude` bloquent l'étape `dotfiles` (`refusing to overwrite`) ; le README (Task 3) doit dire de migrer d'abord. Test : aucun (documentaire), vérifié en revue.
2. Un `HOME` où `~/.claude` n'existe pas encore : `mise dot apply` doit créer les dossiers parents (`~/.claude/skills/`, `~/.config/rtk/`). Couvert par HAR-001 (HOME vide).
3. Un nom de skill avec espace ou point : la cohérence skills ↔ `[dotfiles]` (Task 3) doit lire `config.toml` en TOML (`mise` ou un parseur), pas par grep de ligne. Test : ajouter au contrôle HAR-004 un skill nommé `a.b`.
4. `file_path` relatif ou contenant `..` dans le hook (Task 2) : `realpath -m` doit normaliser avant le test ; un lien pendant (cible inexistante) ne doit pas faire planter le hook. Test unitaire bats : lien pendant vers le main checkout → deny, code de sortie 0.
5. `claude plugin list` qui échoue (non connecté) dans `claude-plugins` (Task 4) : la tâche ne doit pas échouer ; elle affiche un avertissement et continue. Test unitaire bats : `claude` factice dont `plugin list` sort en 1.

---

### Task 1: Import du harness et contrôle d'hygiène — branche `feat/harness-import` (déjà créée, porte la spec et ce plan)

**Criteria:** HAR-006 · **Frontière :** tâche mise

**Files:**
- Create: `claude/{CLAUDE.md,RTK.md,settings.json}`, `claude/agents/{acceptance-writer,implementer,mutant-writer}.md`, `claude/hooks/` (scripts `*.sh`, `mise.toml`, `tests/*.bats`, `tests/helpers.bash`), `claude/skills/<17 skills>/`, `rtk/{config,filters,pipeline}.toml`, `rtk/pipelines/`
- Create: `tasks/check/harness`
- Modify: `.github/workflows/ci.yml` (shellcheck sur `tasks/` récursif ; étape `mise run check:harness`)

**Interfaces:**
- Produces: `mise run check:harness` — code 0 si tout passe, sinon non nul avec une ligne `check:harness: <fichier>: <raison>` par problème. Task 3 y ajoute des contrôles.

- [ ] **Step 1:** copier depuis le poste avec `rsync -a --exclude='*.bak*' --exclude=.archive --exclude=synced` (skills : seulement les dossiers réels, pas les liens `curl-md`/`find-skills`). Vérifier : `find claude rtk -name '*.bak*' -o -type l | wc -l` → `0` ; `ls claude/skills | wc -l` → `17`.
- [ ] **Step 2:** `tasks/check/harness` (`#MISE description="Contrôle le harness versionné"`) : lance `bats claude/hooks/tests/`, puis échoue sur tout `*.bak*` sous `claude/`/`rtk/` et sur tout fichier contenant `/home/arnaud` hors `claude/skills/*/references/`.
- [ ] **Step 3:** `mise run check:harness` → 114 tests bats `ok`, code 0. Ajouter un `claude/x.bak-1` temporaire → code non nul nommant le fichier ; retirer.
- [ ] **Step 4:** CI : étendre shellcheck, ajouter l'étape `mise run check:harness` après « Adopt as global config ».
- [ ] **Step 5:** commit `feat(harness): importer le harness Claude Code et RTK`, PR, CI verte.

### Task 2: Le hook résout les liens — branche `feat/harness-hook-realpath`

**Criteria:** HAR-002 · **Frontière :** entrée JSON / sortie du hook

**Files:**
- Modify: `claude/hooks/require-worktree.sh` (calcul du chemin Edit/Write/NotebookEdit)
- Test (unitaire, Review Focus 4): `claude/hooks/tests/require-worktree.bats`

**Interfaces:**
- Consumes: `check:harness` (Task 1) exécute les bats.
- Produces: rien de nouveau ; même protocole JSON du hook.

- [ ] **Step 1:** résoudre `file_path` avec `realpath -m` avant la remontée `dirname` et le test `in_main_checkout` ; `git check-ignore` porte sur le chemin résolu.
- [ ] **Step 2:** `mise run check:harness` → tous `ok` (anciens + acceptance + Review Focus 4).
- [ ] **Step 3:** commit `fix(hooks): require-worktree résout les liens`, PR, CI verte.

### Task 3: Déploiement par `[dotfiles]` — branche `feat/harness-dotfiles`

**Criteria:** HAR-001, HAR-003, HAR-004 · **Frontière :** CLI mise sur un `HOME` temporaire ; tâche mise

**Files:**
- Modify: `config.toml` section `[dotfiles]` (ligne 102) : une entrée par fichier, par dossier et par skill (liste dans la spec, « Déploiement »), commentaire en tête rappelant « un skill = une ligne ».
- Modify: `tasks/check/harness` : (a) cohérence `claude/skills/*` ↔ entrées `[dotfiles]` et existence de chaque `source` ; (b) `mise dot apply --yes` dans un `HOME` temporaire puis vérification par `realpath` de chaque cible.
- Modify: `README.md` : section « Harness Claude Code » (contenu, ajout d'un skill, migration d'un poste existant avant `mise bootstrap`).

**Interfaces:**
- Consumes: `check:harness` (Task 1).
- Produces: entrées `[dotfiles]` dont la clé est la cible `~/…` et `source` un chemin relatif à la racine du dépôt.

- [ ] **Step 1:** écrire les entrées `[dotfiles]`.
- [ ] **Step 2:** contrôle (a), lecture du TOML structurée (Review Focus 3).
- [ ] **Step 3:** contrôle (b). Point non vérifié : comment faire lire `config.toml` du dépôt à mise sous `HOME=<tmp>` (`MISE_GLOBAL_CONFIG_FILE`, `MISE_CONFIG_DIR`) ; le vérifier d'abord en bac à sable et noter dans le ledger la variable retenue.
- [ ] **Step 4:** `mise run check:harness` → code 0 ; `mise bootstrap --dry-run --yes` en CI reste vert.
- [ ] **Step 5:** README, commit `feat(harness): déployer le harness par mise dotfiles`, PR, CI verte.

### Task 4: Installation du plugin superpowers — branche `feat/harness-plugin`

**Criteria:** HAR-005 · **Frontière :** tâche `claude-plugins`, `claude` simulé

**Files:**
- Create: `tasks/claude-plugins` (`#MISE description="Installe les plugins Claude Code manquants"`)
- Modify: `tasks/bootstrap` (dernière étape : `mise run claude-plugins`)
- Test (unitaire, Review Focus 5): `tests/claude-plugins.bats` (lancé par `check:harness`)

**Interfaces:**
- Consumes: `check:harness` (Task 1).
- Produces: `mise run claude-plugins`, idempotente, liste de plugins en tête du script : `superpowers@anthropic-plugin-directory`.

- [ ] **Step 1:** pour chaque plugin de la liste : sans `claude` sur le `PATH`, ne fait rien ; si `claude plugin list` échoue, avertit et continue ; si l'id est absent de la sortie, `claude plugin install <id>`.
- [ ] **Step 2:** étendre `check:harness` à `tests/*.bats` ; `mise run check:harness` → code 0.
- [ ] **Step 3:** commit `feat(bootstrap): installer les plugins Claude Code`, PR, CI verte.
