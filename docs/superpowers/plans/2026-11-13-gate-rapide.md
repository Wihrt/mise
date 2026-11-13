# Porte verte plus rapide — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `harness gate green --proofs` juge la preuve de code minimal en parallèle sous `--jobs` et ne juge plus les blocs de documentation ni de commentaires.

**Architecture:** La boucle `runMinimal` est découpée en trois temps : lister les blocs, les juger (en place sans `--jobs`, dans des worktrees jetables avec `--jobs`, résultats mis en cache comme `r.judged` pour les mutants), imprimer les verdicts dans l'ordre des blocs. Les règles d'exclusion sont des fonctions pures du domaine `tdd`, appelées avant tout jugement.

**Tech Stack:** Go (module `harness/`), tests d'acceptation Go qui lancent le binaire (`harness/acceptance`), `mise run test`, exécution par `stacked-tdd-mr`.

**Spec:** `claude/docs/specs/2026-11-13-gate-rapide-design.md`

## Global Constraints

- Une tâche = un slug = une branche = une MR, empilées sur `feat/gate-rapide-spec` (PR #80). Préfixe d'ID : `PRV`.
- Frontière de tous les critères : la ligne de commande `harness gate green --proofs` sur un dépôt git jetable, sortie standard, code de sortie et durée de la commande.
- Aucun code de test d'acceptation dans ce plan : l'`acceptance-writer` l'écrit depuis les critères de la spec, avec les helpers de `harness/acceptance/helpers_test.go`.
- Sans `--jobs` (ou `--jobs 1`) et sans bloc exclu, la sortie de la porte est identique à celle du commit `6030846`.
- Verdict d'un bloc inchangé : `UNNEEDED` quand la tâche sort avec le code 0 et un rapport `tdd.ReportOK`, nécessaire sinon (délai dépassé compris).
- Liste intégrée, valeurs exactes : `*.md`, `*.rst`, `*.adoc`. Ligne exacte : `MINIMAL EXCLUDED <fichier>:<début>-<fin>`.
- Le fichier de `--spec` reste sauté sans aucune ligne (PRV-014).
- Couches : les règles pures vont dans `internal/domain/tdd`, l'orchestration dans `internal/app/gate`, les drapeaux dans `cmd/harness/gate.go` (`lint:go` vérifie les couches).
- Commits conventionnels, sans aucune attribution IA. Sous-agents en `sonnet`.
- Porte qui juge : le binaire installé `~/.local/bin/harness`, pas celui de la pile.
- Tâche mise jugée : `test` (`Commande : mise run test` dans `claude/TESTS.md`). Dépôt donné à `--repo` : le dossier qui contient `TESTS.md`, soit `<worktree>/claude`.

## Review Focus

Cas que la spec écrit et qu'aucun critère ne porte seul ; chacun a son test unitaire dans la tâche qui possède le code.

- Un bloc exclu qui est aussi nommé par `--unneeded-ok` : il imprime `MINIMAL EXCLUDED`, sans erreur (tâche 2).
- Un bloc de commentaires indenté : les espaces de tête ne comptent pas (tâche 3).
- Sous `--jobs`, des blocs exclus mêlés à des blocs jugés : les lignes sortent dans l'ordre des blocs (tâche 2).
- Sous `--jobs`, une tâche qui dépasse son délai pour un bloc : le bloc est dit nécessaire, la porte continue (tâche 1).
- Sous `--jobs`, une erreur d'environnement dans un worktree (`mise` introuvable) : la porte sort en erreur, code 2, et les worktrees sont retirés (tâche 1).

---

### Task 1: `minimal-parallele` — le code minimal se juge en parallèle

**Critères :** PRV-020, PRV-021. **Mode :** atdd. **Branche :** `feat/minimal-parallele` (base `feat/gate-rapide-spec`).

**Files:**
- Modify: `harness/internal/app/gate/proofs.go:113-145` (`judgeInWorktree`), `:283-320` (`runMinimal`)
- Modify: `harness/internal/app/gate/gate.go:66-71` (struct `run`)
- Modify: `harness/cmd/harness/gate.go:99` (aide de `--jobs`)
- Modify: `claude/skills/stacked-tdd-mr/SKILL.md` (étapes 4b et 8)
- Test: `harness/acceptance/minimal_parallele_test.go` (acceptance-writer), `harness/internal/app/gate/gate_test.go` (unitaires, libres)

**Interfaces:**
- Consumes: `tdd.AddedBlocks`, `Gate.worktrees`, `run.runTask`, `run.restoreTree`.
- Produces:
  - `func blockTag(b tdd.Block) string` : `<fichier>:<début>-<fin>`.
  - `func (r *run) minimalBlocks(repo string) []tdd.Block` : blocs ajoutés entre `--since` (défaut `--test`) et `--feat`, fichier de `--spec` retiré, dans l'ordre du diff.
  - `func (r *run) removeBlock(dir string, b tdd.Block) error` : retire les lignes du bloc dans `dir/<b.Path>`.
  - `func (r *run) withWorktree(repo string, fn func(wt string) error) error` : worktree jetable de `HEAD`, ajout et retrait sous `Gate.worktrees` ; `judgeInWorktree` l'utilise aussi.
  - `type blockJudgement struct { unneeded bool; lines []string }` et `run.blocks map[string]blockJudgement`, clé `blockTag`.
  - `func (r *run) judgeBlocks(repo string, blocks []tdd.Block) error` : remplit `run.blocks`, `Jobs` blocs à la fois, un worktree par bloc.
  - `func (r *run) blockUnneeded(repo string, b tdd.Block) (bool, error)` : rejoue les lignes du cache quand le bloc y est, sinon juge en place comme aujourd'hui.

- [ ] **Step 1 : acceptation.** `acceptance-writer` écrit `harness/acceptance/minimal_parallele_test.go` et la section `minimal-parallele` de `claude/TESTS.md` (commit `test(minimal-parallele): acceptance`). Porte rouge : `harness gate red --slug minimal-parallele --ac PRV-020,PRV-021`, attendu `PASS`.
- [ ] **Step 2 : découper `runMinimal` sans changer la sortie.** Extraire `blockTag`, `minimalBlocks`, `removeBlock`, `blockUnneeded` ; `runMinimal` devient : lister, puis pour chaque bloc `blockUnneeded` et la ligne `UNNEEDED` / `UNNEEDED ACCEPTED`. Vérifier : `mise run test:go`, tous les tests existants verts (PRV-009, PRV-013, PRV-014, PRV-018).
- [ ] **Step 3 : `withWorktree`.** L'extraire de `judgeInWorktree`, qui l'appelle. Vérifier : `mise run test:go`, PRV-019 vert.
- [ ] **Step 4 : `judgeBlocks`.** Quand `r.o.Jobs > 1`, `runMinimal` l'appelle avant la boucle d'impression. Chaque bloc : `withWorktree`, `removeBlock` dans le worktree, `runTask` dans le worktree avec un `run` dont `lines` capture la sortie. Une erreur de `runTask` remonte telle quelle (code 2) ; un délai dépassé donne `unneeded: false`.
- [ ] **Step 5 : tests unitaires du Review Focus** dans `gate_test.go` avec les ports factices existants : délai dépassé sous `--jobs` (bloc nécessaire, la porte continue) ; erreur de `Tasks.Run` sous `--jobs` (code 2, autant de `worktree remove` que de `worktree add`).
- [ ] **Step 6 : vérifier.** `mise run test:go` : PRV-020 et PRV-021 verts. `mise run check` vert.
- [ ] **Step 7 : commit** `feat(minimal-parallele): le code minimal se juge en parallèle sous --jobs`.
- [ ] **Step 8 : compétence.** Dans `stacked-tdd-mr/SKILL.md` : l'étape 4b dit que `--jobs` couvre aussi la preuve de code minimal ; l'étape 8 lance `green` avec `--jobs <n>`. Commit `docs(minimal-parallele): --jobs couvre la preuve de code minimal`.
- [ ] **Step 9 : porte verte** (`stacked-tdd-mr`, étapes 4 et 4b), puis MR vers `feat/gate-rapide-spec`.

### Task 2: `minimal-doc` — la documentation n'est pas jugée

**Critères :** PRV-022, PRV-023. **Mode :** atdd. **Branche :** `feat/minimal-doc` (base `feat/minimal-parallele`).

**Files:**
- Modify: `harness/internal/domain/tdd/proofs.go` (à côté de `Block`)
- Modify: `harness/internal/app/gate/gate.go:15-38` (`Options`), `harness/internal/app/gate/proofs.go` (`runMinimal`)
- Modify: `harness/cmd/harness/gate.go:22` (synopsis), `:109` (drapeaux de `green`)
- Modify: `claude/skills/stacked-tdd-mr/SKILL.md` (étape 4)
- Test: `harness/acceptance/minimal_doc_test.go` (acceptance-writer), `harness/internal/domain/tdd/tdd_test.go`, `harness/internal/app/gate/gate_test.go` (unitaires, libres)

**Interfaces:**
- Consumes (tâche 1): `minimalBlocks`, `blockTag`, `judgeBlocks`, `blockUnneeded`.
- Produces:
  - `var DocPatterns = []string{"*.md", "*.rst", "*.adoc"}` dans `tdd`.
  - `func IsDocPath(path string, extra []string) bool` dans `tdd` : `Fnmatch` du chemin contre `DocPatterns` puis `extra`.
  - `Options.MinimalExclude []string` ; drapeau `--minimal-exclude <motif>`, répétable (`list{&o.MinimalExclude}`), mode `green` seulement.
  - `func (r *run) excludedBlock(repo string, b tdd.Block) bool` dans `gate` : la tâche 3 l'étend.

- [ ] **Step 1 : acceptation.** `acceptance-writer` écrit `harness/acceptance/minimal_doc_test.go` et la section `minimal-doc` de `claude/TESTS.md`. Porte rouge : `--ac PRV-022,PRV-023`, attendu `PASS`.
- [ ] **Step 2 : domaine.** `DocPatterns` et `IsDocPath`, avec leur test unitaire (`docs/guide.md` et `README.rst` vrais ; `scripts/a.py` faux ; `NOTICE` vrai avec `extra = ["NOTICE"]`).
- [ ] **Step 3 : drapeau.** `Options.MinimalExclude`, `--minimal-exclude` dans `cmd/harness/gate.go`, synopsis mis à jour.
- [ ] **Step 4 : `excludedBlock` dans `runMinimal`.** Un bloc exclu n'est donné ni à `judgeBlocks` ni à `blockUnneeded` ; à son tour dans la boucle d'impression il imprime `MINIMAL EXCLUDED <blockTag>` et ne change pas le verdict de la porte.
- [ ] **Step 5 : tests unitaires du Review Focus** dans `gate_test.go` : bloc exclu aussi nommé par `--unneeded-ok` (`MINIMAL EXCLUDED`, pas d'erreur) ; sous `--jobs 2`, blocs exclu, jugé, exclu : lignes dans l'ordre des blocs.
- [ ] **Step 6 : vérifier.** `mise run test:go` : PRV-022 et PRV-023 verts. `mise run check` vert.
- [ ] **Step 7 : commit** `feat(minimal-doc): la documentation n'est pas jugée par la preuve de code minimal`.
- [ ] **Step 8 : compétence.** `stacked-tdd-mr/SKILL.md`, étape 4 : `MINIMAL EXCLUDED <file>:<start>-<end>` rejoint la liste des lignes ; `--minimal-exclude <pattern>` rejoint la ligne de commande de `green`. Commit `docs(minimal-doc): MINIMAL EXCLUDED et --minimal-exclude`.
- [ ] **Step 9 : porte verte**, puis MR vers `feat/minimal-parallele`.

### Task 3: `minimal-commentaires` — les commentaires ne sont pas jugés

**Critères :** PRV-024. **Mode :** atdd. **Branche :** `feat/minimal-commentaires` (base `feat/minimal-doc`).

**Files:**
- Modify: `harness/internal/domain/tdd/proofs.go`
- Modify: `harness/internal/app/gate/proofs.go` (`excludedBlock`)
- Modify: `claude/skills/stacked-tdd-mr/SKILL.md` (étape 4)
- Test: `harness/acceptance/minimal_commentaires_test.go` (acceptance-writer), `harness/internal/domain/tdd/tdd_test.go` (unitaire, libre)

**Interfaces:**
- Consumes (tâche 2): `excludedBlock`.
- Produces: `func CommentOnly(lines []string) bool` dans `tdd` : vrai quand chaque ligne, espaces de tête et de fin retirés, est vide ou commence par `#` ou `//`.

- [ ] **Step 1 : acceptation.** `acceptance-writer` écrit `harness/acceptance/minimal_commentaires_test.go` et la section `minimal-commentaires` de `claude/TESTS.md`. Porte rouge : `--ac PRV-024`, attendu `PASS`.
- [ ] **Step 2 : domaine.** `CommentOnly`, avec son test unitaire (`["# a", "", "  # b"]` vrai ; `["// a"]` vrai ; `["# a", "x = 1"]` faux).
- [ ] **Step 3 : `excludedBlock`** lit les lignes `b.Start` à `b.End` de `repo/<b.Path>` et rend vrai aussi quand `tdd.CommentOnly` est vrai.
- [ ] **Step 4 : vérifier.** `mise run test:go` : PRV-024 vert. `mise run check` vert.
- [ ] **Step 5 : commit** `feat(minimal-commentaires): un bloc de commentaires n'est pas jugé`.
- [ ] **Step 6 : compétence.** `stacked-tdd-mr/SKILL.md`, étape 4 : `--unneeded-ok` ne cite plus le commentaire comme cas d'emploi ; il reste pour une docstring, un commentaire `/* */` ou un message d'erreur qu'aucun test ne lit. Commit `docs(minimal-commentaires): --unneeded-ok ne sert plus aux commentaires`.
- [ ] **Step 7 : porte verte**, puis MR vers `feat/minimal-doc`.

## Fin de pile

`stacked-tdd-mr`, étape 8 : preuve de code minimal une fois sur `feat/minimal-commentaires` avec `--since <premier commit d'acceptation de la pile>`, revue finale, CI de toutes les MR, rapport d'état.

## Point à constater à la première porte

`runMinimal` lit `repo + "/" + b.Path` (`proofs.go:295`), où `b.Path` vient de `git diff`, donc relatif à la racine du dépôt, alors que `--repo` est ici `<worktree>/claude` (`TESTS.md` y est, `gate.go:258`). Sur ce dépôt, la preuve de code minimal de l'étape 8 lirait donc `claude/harness/...`, qui n'existe pas. Non vérifié par exécution. Si la porte sort en erreur à l'étape 8 : arrêt et décision de l'utilisateur, aucun contournement dans la pile.
