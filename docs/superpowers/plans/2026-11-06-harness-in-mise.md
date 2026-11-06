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
- [ ] **Step 4:** commit `chore(tooling): outillage du dépôt et import du harness`.
- [ ] **Step 5 (généralisation, dépôt public) :** retirer du dépôt `claude/skills/*/references/evidence-*.md`, `claude/skills/*/.ledger.jsonl`, `claude/skills/*/.sidecar.json.lock` et les ignorer dans `.gitignore` ; dans `forge-glab`, `forge-gh`, `stacked-tdd-mr` : aucun hôte en dur, détection `github.com` → gh, sinon hôte d'`origin` connu de `glab auth status` → glab, ou de `gh auth status --hostname <hôte>` → gh, sinon arrêt ; dans `helm-chart-authoring`, `helm-render-diff-check`, `parallel-repo-plan-dispatch` : noms de projets clients remplacés par des termes génériques, sens inchangé. Vérifier : `git grep -ciE '<motifs internes>'` → 0 (motifs fournis hors dépôt). Commit `refactor(harness): skills sans hôte ni projet interne`, PR vers `main`, CI verte.

### Task 1: Hygiène du harness — branche `feat/harness-hygiene`, slug `harness-hygiene`, préfixe `HAR`

**Criteria:** HAR-006 · **Frontière :** tâche mise `lint:harness`

**Files:**
- Modify: `.config/mise.toml` (tâche `lint:harness`, script `.config/lint-harness.sh` si plus de 3 lignes)

**Interfaces:**
- Produces: `mise run lint:harness` — code 0 si propre, sinon non nul avec une ligne `lint:harness: <fichier>: <raison>` par problème. Task 3 y ajoute la cohérence.

- [ ] **Step 1:** `lint:harness` échoue sur tout `*.bak*` sous `claude/`/`rtk/`, tout fichier contenant `/home/arnaud` hors `claude/skills/*/references/`, et tout fichier contenant un motif de `.config/harness-denylist.local` (gitignoré ; nom du fichier affiché, jamais le motif). `.config/harness-denylist.local` absent : ligne d'avertissement, contrôle ignoré.
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

### Task 5: Correctifs des remarques Copilot — branche `feat/harness-copilot-fixes`, slug `harness-copilot-fixes`

**Criteria:** HAR-007, HAR-008, HAR-009, HAR-010, HAR-011, HAR-012 · **Frontières :** voir chaque critère

**Files:**
- Modify: `claude/hooks/agent-guard.sh`, `claude/hooks/never-merge.sh`
- Modify: `.config/mise.toml` (tâche de scan des secrets indexés appelée par le hook pre-commit `lint-secrets` ; `lint:secrets` garde le scan d'historique pour la CI), `.pre-commit-config.yaml`, `claude/skills/repo-tooling/templates/` (même correction dans les gabarits), `claude/skills/repo-tooling/SKILL.md`
- Modify: `.config/lint-harness.sh`
- Modify: `claude/skills/k8s-restricted/policy/pss-restricted.rego`, `claude/skills/helm-render-diff-check/scripts/policy-rules.py`
- Sans test (docs) : `claude/skills/k8s-restricted/SKILL.md` ligne 26 (ResourceQuota : une déclaration requise par un quota CPU/mémoire omise fait refuser le Pod à l'admission ; seul un quota sur une ressource non exigée, comme ephemeral-storage, tolère l'omission)
- Sans test (outillage) : `test:hooks` et `test:acceptance` écrivent chacun leur rapport (`test:hooks` → `.tdd/hooks.xml`, `test:acceptance` → `.tdd/junit.xml`, seul lu par `tdd-gate`)

## Critères d'acceptation

### HAR-007 — L'auteur des tests ne peut pas sortir des dossiers de test par `..`, par un lien ou hors du dépôt
- **Étant donné** le rôle `acceptance-writer` et un dépôt avec `pkg/calc.go` et `pkg/tests/`, dont un lien `pkg/tests/link.go` vers `pkg/calc.go`
- **Quand** `agent-guard.sh` reçoit un `Edit` dont `file_path` vaut `pkg/tests/../calc.go`, `pkg/tests/link.go`, ou un chemin absolu hors du répertoire de travail de l'appel (par exemple `<autre dossier>/tests/a_test.go` ou `<autre dossier>/TESTS.md`)
- **Alors** il refuse ; il accepte `pkg/tests/calc_test.go` (fichier ordinaire sous `tests/`)
- **Frontière** : entrée JSON / sortie du hook

### HAR-008 — Un push forcé reste refusé même accompagné d'un lease
- **Étant donné** le hook `never-merge.sh`
- **Quand** il reçoit `git push -f --force-with-lease origin feat`, ou `git push --force --force-with-lease origin feat`
- **Alors** il refuse ; il accepte `git push --force-with-lease origin feat`
- **Frontière** : entrée JSON / sortie du hook

### HAR-009 — Le hook pre-commit détecte un secret indexé
- **Étant donné** un dépôt git temporaire où un fichier contenant un jeton d'accès factice au format GitHub (`ghp_` + 36 caractères alphanumériques) est indexé mais pas commité
- **Quand** la tâche mise appelée par le hook pre-commit `lint-secrets` est lancée dans ce dépôt
- **Alors** elle échoue (code non nul) ; sans ce fichier indexé elle réussit
- **Frontière** : tâche mise du hook pre-commit

### HAR-010 — Les fichiers binaires ne contournent pas `lint:harness`, et le chemin personnel n'est jamais affiché
- **Étant donné**, sous `claude/`, un fichier contenant un octet NUL et `/home/arnaud`, ou un octet NUL et un motif de `.config/harness-denylist.local`
- **Quand** `mise run lint:harness` est lancé
- **Alors** il échoue en nommant le fichier ; sa sortie ne contient ni `/home/arnaud` ni le motif
- **Frontière** : tâche mise

### HAR-011 — La politique restricted refuse un Pod nu et un UID 0
- **Étant donné** un manifeste `kind: Pod` (sans contrôleur) avec `hostNetwork: true`, ou un Deployment dont le conteneur a `runAsNonRoot: true` et `runAsUser: 0`
- **Quand** `conftest test --policy claude/skills/k8s-restricted/policy <manifeste>` est lancé
- **Alors** il échoue (code non nul) avec au moins un refus ; un Pod nu conforme au profil restricted passe
- **Frontière** : CLI conftest

### HAR-012 — Le rapport des règles réseau montre les entités Cilium
- **Étant donné** une CiliumNetworkPolicy dont une règle d'egress a `toEntities: [world]` et une règle d'ingress `fromEntities: [cluster]`
- **Quand** `python3 -I claude/skills/helm-render-diff-check/scripts/policy-rules.py <fichier>` est lancé
- **Alors** sa sortie nomme `world` et `cluster` parmi les pairs, et ne présente pas ces règles comme sans pair (`none`)
- **Frontière** : CLI du script
