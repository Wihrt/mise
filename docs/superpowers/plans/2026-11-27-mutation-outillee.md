# Mutation outillée — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: exécuter avec `stacked-tdd-mr` (wrapper sur superpowers:subagent-driven-development). Une tâche = une branche = une MR, empilées. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remplacer la preuve TDD du harnais (porte red/green, `mutant-writer`, `TESTS.md`) par une tâche `mutation` déclarée dans chaque dépôt, qui s'appuie sur gremlins, pytest-gremlins et helm-mutation-test, et écrire la règle des scripts.

**Architecture:** `repo-tooling` livre trois fragments TOML (`mutation-go.toml`, `mutation-py.toml`, `mutation-helm.toml`). Le verdict de chaque tâche est rendu par `jq` sur le rapport de l'outil. `check` lance `mutation`. Le harnais ne garde que `gate coverage`, et les skills et agents perdent les étapes de preuve.

**Tech Stack:** mise, gremlins 0.6.0 (`github:` backend), pytest-gremlins 1.15.0 (groupe dev uv), helm-mutation-test 1.2.0 (`github:` backend), jq, Go 1.27 (harnais), bats.

**Spec:** `claude/docs/specs/2026-11-27-mutation-outillee-design.md` ; ADR `docs/adr/0003-la-preuve-des-tests-est-une-mutation-outillee-dans-chaque-depot.md`.

## Global Constraints

- Versions exactes : `"github:go-gremlins/gremlins" = "0.6.0"`, `"github:mmpyro/helm-mutation-test" = "1.2.0"`, `pytest-gremlins==1.15.0` dans `[dependency-groups] dev`, `jq` épinglé (`mise ls-remote jq`).
- Le verdict ne vient jamais du code de sortie des outils : les seuils natifs de gremlins sortent en 0 (essai du 2026-11-27), et pytest-gremlins sort en 0 avec des survivants.
- Une tâche échoue dès qu'un mutant du code changé depuis `MUTATION_BASE` survit ou n'est atteint par aucun test ; elle sort en succès quand rien n'a changé.
- `MUTATION_BASE = "origin/main"` dans `[env]`. La CI le remplace par `origin/<branche cible>` (`origin/${{ github.base_ref }}` sur GitHub).
- Les tâches `mutation:*` restent hors de `test:*`. `check` dépend de `lint`, `test` et `mutation`.
- Les tâches séquencent des commandes. La seule garde permise est `test -n "$targets" || exit 0` de `mutation:py`. Toute autre décision passe par `jq`, ou par du Go ou du Python testé.
- Ordre des commits d'une branche (skill actuel) : `docs(spec)` → `chore(tooling)` → `test(<slug>): acceptance` → `feat(<slug>)`.
- Cette pile s'exécute avec le `stacked-tdd-mr` **actuel** et le binaire `~/.local/bin/harness` actuel : ses tâches passent encore par `gate red`/`green`.

## Review Focus

1. **`mutation:go` sur le module `harness/` de ce dépôt.** Le module n'a jamais été muté. Toute ligne Go ajoutée ou modifiée dans la pile (tâche 5 : `IsTestPath` déplacé) doit être tuée par un test unitaire du même paquet. Sinon, la CI de la MR échoue. La tâche 5 porte ce test.
2. **Diff vide.** Une MR qui ne touche aucun fichier Go, Python ou de chart doit laisser `mutation` vert, y compris en Helm. Les clauses « rien n'a changé » de MUT-002 et MUT-003 le couvrent. Pour Helm, la tâche 3 ajoute un test d'acceptance sur un diff vide.
3. **`MUTATION_BASE` absent en local.** Une branche locale sans `origin/main` à jour fait diverger le diff. Le skill dit de lancer `git fetch origin` avant `mise run mutation`, et la tâche 4 l'écrit.
4. **Fichier Python supprimé ou renommé.** `--diff-filter=AMR` exclut les suppressions. La tâche 2 ajoute un test d'acceptance où un fichier du paquet est supprimé depuis `MUTATION_BASE` : la tâche doit sortir en succès.
5. **Hooks qui nomment `mutant-writer`.** `harness/internal/app/agentguard.go` et `claude/hooks/tests/agent-guard.bats` ne doivent ni refuser ni exiger un agent qui n'existe plus. La tâche 5 met à jour ces deux fichiers.

---

### Task 1: Fragment `mutation-go` et `mutation:go` sur le harnais

**Critères :** MUT-002. **Mode :** atdd. **Tâche de test :** `test:go`.

**Files:**
- Create: `claude/skills/repo-tooling/templates/mutation-go.toml`, qui contient la partie commune (`jq`, `MUTATION_BASE`, `[tasks.mutation]`, `check`) et `mutation:go`, tels qu'écrits dans la section « Boîte à outils » de la spec.
- Modify: `claude/skills/repo-tooling/SKILL.md` : une section « Mutation » qui dit quel fragment fusionner selon la pile, que le verdict est le `jq`, que les tests unitaires sont la cible, et que la CI passe `MUTATION_BASE`.
- Modify: `.config/mise.toml` : `[tools]` gremlins 0.6.0 et jq, `[env] MUTATION_BASE`, `[tasks."mutation:go"]` avec `dir = "harness"`, `[tasks.mutation]`, et `check` qui dépend de `mutation`.
- Modify: `.github/workflows/ci.yml` : `env: MUTATION_BASE: origin/${{ github.base_ref }}` sur le job qui lance `mise run check` (`fetch-depth: 0` est déjà là, ligne 41).
- Test : les tests d'acceptance de l'`acceptance-writer` (dépôt Go jetable construit avec le fragment, lancé par `mise run mutation:go`).

**Interfaces:**
- Produces: le fichier `templates/mutation-go.toml` et sa partie commune, que les tâches 2 et 3 réutilisent sans la dupliquer (leurs fragments ne contiennent que leur propre `[tools]` et leur propre tâche) ; la variable `MUTATION_BASE` ; la tâche agrégée `mutation`.

- [ ] **Step 1:** l'`acceptance-writer` écrit les tests de MUT-002 et commite `test(mutation-go): acceptance`. Porte red.
- [ ] **Step 2:** écrire le fragment, la section du skill, et le câblage `.config/mise.toml` et `ci.yml`.
- [ ] **Step 3:** lancer `mise run mutation:go` à la racine. Attendu : code 0 (aucun fichier Go changé depuis `origin/main`).
- [ ] **Step 4:** porte green, puis `mise run check`. Attendu : code 0.
- [ ] **Step 5:** commiter `feat(mutation-go): ...`.

### Task 2: Fragment `mutation-py`

**Critères :** MUT-003. **Mode :** atdd. **Tâche de test :** `test:go` (les tests d'acceptance du gabarit vivent avec ceux de la tâche 1).

**Files:**
- Create: `claude/skills/repo-tooling/templates/mutation-py.toml`, avec `mutation:py` tel qu'écrit dans la spec. `<paquet>` reste un paramètre que le skill dit de remplacer.
- Modify: `claude/skills/repo-tooling/SKILL.md` (section « Mutation ») : `pytest-gremlins==1.15.0` va dans le groupe dev, `[tool.pytest-gremlins] paths`, la cible est `tests/unit`, le cache CI se garde avec `if: always()`.
- Modify: `claude/skills/stack-py-app/SKILL.md` : les tests unitaires sont obligatoires et c'est `mutation:py` qui les mesure.

**Interfaces:**
- Consumes: la partie commune de `mutation-go.toml` (tâche 1).

- [ ] **Step 1:** l'`acceptance-writer` écrit les tests de MUT-003, plus le cas « fichier supprimé » de la Review Focus 4. Il commite `test(mutation-py): acceptance`. Porte red.
- [ ] **Step 2:** écrire le fragment et les deux textes de skill.
- [ ] **Step 3:** porte green, puis `mise run check`. Attendu : code 0.
- [ ] **Step 4:** commiter `feat(mutation-py): ...`.

### Task 3: Fragment `mutation-helm`

**Critères :** MUT-004. **Mode :** atdd. **Tâche de test :** `test:go`.

**Files:**
- Create: `claude/skills/repo-tooling/templates/mutation-helm.toml`, avec `mutation:helm:<chart>` tel qu'écrit dans la spec.
- Modify: `claude/skills/repo-tooling/SKILL.md` (section « Mutation ») : une tâche par chart, le binaire autonome sans `helm plugin install`, et `-f 'tests/**/*_test.yaml'` quand les suites sont dans des sous-répertoires.
- Modify: `claude/skills/helm-chart-authoring/SKILL.md` : une ligne qui renvoie à `mutation:helm:<chart>`.

**Interfaces:**
- Consumes: la partie commune de `mutation-go.toml` (tâche 1).

- [ ] **Step 1:** l'`acceptance-writer` écrit les tests de MUT-004, plus le cas « diff vide » de la Review Focus 2. Il commite `test(mutation-helm): acceptance`. Porte red.
- [ ] **Step 2:** écrire le fragment et les deux textes de skill.
- [ ] **Step 3:** porte green, puis `mise run check`. Attendu : code 0.
- [ ] **Step 4:** commiter `feat(mutation-helm): ...`.

### Task 4: Skills, agents et règles sans preuve TDD

**Critères :** MUT-005. **Mode :** atdd. **Tâche de test :** `test:dotfiles`.

**Files:**
- Delete: `claude/agents/mutant-writer.md`, `claude/TESTS.md` et `claude/skills/stacked-tdd-mr/templates/TESTS.md`. Retirer leur entrée de `config.toml` `[dotfiles]`.
- Modify: `claude/skills/stacked-tdd-mr/SKILL.md`. Réécrire le flux de la section « Flux `stacked-tdd-mr` après le changement » de la spec et retirer les étapes 3, 4 (2), 4b et 5c, le mode réécriture/`replay`, `CLAUSES`, `TESTS-CHANGE` et `TESTS.md`. Le flux local est `git fetch origin && MUTATION_BASE=<branche N-1> mise run check`.
- Modify: `claude/agents/acceptance-writer.md` (plus de section `TESTS.md`) et `claude/agents/implementer.md` (il écrit les tests unitaires, et les mutants survivants ou non couverts sont ses constats).
- Modify: `claude/CLAUDE.md`, section Development :
  - la phrase « Acceptance tests … proven red then green by `harness gate` and listed in `TESTS.md` » devient : la preuve est `mise run mutation` en CI, plus la porte de couverture ;
  - « Unit tests are free scaffolding » devient « Unit tests are required: they are what `mise run mutation` measures » ;
  - « Direct fix » perd `TESTS.md` ;
  - ajouter la règle des scripts, avec le texte exact de la spec.
- Modify: `claude/skills/repo-tooling/SKILL.md`, avec la même règle des scripts.
- Modify: toute autre référence trouvée par `grep -rlE 'mutant-writer|TESTS\.md|gate (red|green|characterize|replay)|CLAUSES|TESTS-CHANGE' claude README.md _typos.toml .gitattributes` hors `claude/docs/specs`. Liste connue au 2026-11-27 :
  - `README.md`, `_typos.toml`, `.gitattributes` ;
  - `claude/docs/regles-generiques.md` ;
  - les skills `forge-gh`, `forge-glab`, `hexagonal-go`, `hexagonal-py`, `opentofu-local`, `stack-go-app` ;
  - `stacked-tdd-mr/templates/acceptance-criteria.md`, `stacked-tdd-mr/templates/socle-application.md` ;
  - `repo-tooling/templates/_typos.toml`, `repo-tooling/templates/gitattributes` ;
  - l'en-tête des fichiers `claude/hooks/tests/{merge-stack,merge-stack-cible,merge-stack-forge,merge-stack-gitlab,never-merge-stack,search-hint}.bats`, qui renvoie aujourd'hui à « Criteria: claude/TESTS.md, section … ». Il renverra à la spec de la section, sous `claude/docs/specs/`.

- [ ] **Step 1:** l'`acceptance-writer` écrit les tests de MUT-005 et commite `test(sans-preuve-tdd): acceptance`. Porte red.
- [ ] **Step 2:** faire les suppressions et les réécritures.
- [ ] **Step 3:** relancer le grep ci-dessus. Attendu : aucune ligne.
- [ ] **Step 4:** porte green, puis `mise run check`. Attendu : code 0.
- [ ] **Step 5:** commiter `feat(sans-preuve-tdd): ...`.

### Task 5: Le harnais ne garde que `gate coverage`

**Critères :** MUT-001. **Mode :** atdd. **Tâche de test :** `test:go` et `test:hooks`.

**Files:**
- Modify: `harness/cmd/harness/gate.go` : `parseGate` ne connaît plus que `coverage` ; `red`, `green`, `characterize` et `replay` tombent dans le cas « sous-commande inconnue » existant.
- Delete: `harness/internal/app/gate/{gate,green,proofs,characterize,replay}.go` et leurs tests ; `harness/internal/domain/tdd/` en entier.
- Move: `IsTestPath(path string) bool` vers `harness/internal/domain/coverage/testpath.go`, avec le même comportement ; `harness/internal/app/gate/coverage.go:62` l'appelle là. Ajouter un test unitaire `testpath_test.go` dans le paquet `coverage`, qui vérifie un cas vrai et un cas faux pour chaque convention que reconnaît la fonction actuelle (Go `_test.go`, Python `test_*.py` et `tests/`, …). C'est la Review Focus 1 : sans ce test, `mutation:go` laisse des mutants non couverts.
- Delete: `harness/acceptance/{red,green,characterize,clauses,tests_gardes,preuves_green,rejeu,mutant_mise,necessaire,parallel,forme,branchement}_test.go`, pour ceux qui ne testent que les commandes retirées. Lire chaque fichier avant de le supprimer : un fichier qui teste aussi `gate coverage` garde ces tests-là. `couverture_test.go` et `local_test.go` restent.
- Modify: `harness/internal/app/agentguard.go`, `harness/internal/app/policies_test.go` et `claude/hooks/tests/agent-guard.bats` : plus de règle propre à `mutant-writer`.
- Modify: `harness/cmd/harness/main.go` : l'aide ne cite plus les commandes retirées.

**Interfaces:**
- Produces: `coverage.IsTestPath(path string) bool`.

- [ ] **Step 1:** l'`acceptance-writer` écrit les tests de MUT-001 et commite `test(gate-coverage-seul): acceptance`. Porte red.
- [ ] **Step 2:** déplacer `IsTestPath` et écrire son test unitaire. `go test ./internal/domain/coverage/` doit passer.
- [ ] **Step 3:** faire les suppressions et les modifications ci-dessus. `go build ./...` et `golangci-lint run ./...` dans `harness/` doivent sortir en 0.
- [ ] **Step 4:** `MUTATION_BASE=<branche de la tâche 4> mise run mutation:go`. Attendu : code 0, `mutants_lived == 0`, `mutants_not_covered == 0`.
- [ ] **Step 5:** porte green, puis `mise run check`. Attendu : code 0.
- [ ] **Step 6:** commiter `feat(gate-coverage-seul): ...`.

### Task 6: `mutation:go` sur un module Go hors de la racine du dépôt (correctif)

**Critères :** MUT-006 (ci-dessous). **Mode :** atdd. **Tâche de test :** `test:go`.

Cause prouvée pendant la tâche 5 : avec `dir = "harness"`, `git diff` rend des chemins préfixés par `harness/`, alors que gremlins attend des chemins relatifs au module. Le run donne `Skipped: 516, Killed: 0`, et la tâche sort en 0 quel que soit le code. Avec `diff.relative=true` (`GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=diff.relative GIT_CONFIG_VALUE_0=true`), le même run donne `Killed: 8, Lived: 0, Not covered: 0`.

**Files:**
- Modify: `claude/skills/repo-tooling/templates/mutation-go.toml`, `[tasks."mutation:go"]` : la garde et gremlins voient le diff relatif au répertoire de la tâche (`diff.relative=true`, par `git -c` pour la garde et par l'environnement pour gremlins). À la racine, cela ne change rien.
- Modify: `.config/mise.toml`, `[tasks."mutation:go"]` : le même réglage.
- Modify: `claude/skills/repo-tooling/SKILL.md` (section « Mutation ») : un module Go hors de la racine prend `dir = "<module>"`.

## Critères d'acceptation

### MUT-006 — `mutation:go` mute le code changé d'un module Go placé dans un sous-répertoire
- **Étant donné** un dépôt jetable dont le module Go est dans `svc/` (cas réel : `harness/go.mod` de ce dépôt), le fragment `mutation-go.toml` avec `dir = "svc"` sur `mutation:go`, et une fonction ajoutée dans `svc/` depuis `MUTATION_BASE`
- **Quand** on lance `mise run mutation:go`
- **Alors**
  - si le test unitaire appelle la fonction sans vérifier son résultat, la tâche sort en échec
  - si le test unitaire vérifie le résultat, la tâche sort en succès

### Task 7: mode complet de la mutation, de temps en temps

**Critères :** MUT-007 (spec). **Mode :** atdd. **Tâche de test :** `test:go`. Spec : section « Mode complet, de temps en temps ».

**Files:**
- Modify: `claude/skills/repo-tooling/templates/mutation-{go,py,helm}.toml` : si `MUTATION_FULL` vaut `1`, la tâche saute la garde et le diff (Go : `gremlins unleash` sans `--diff` ; Python : `targets=$(git ls-files '<paquet>/*.py' | paste -sd, -)` ; Helm : sans `--since`). Le verdict `jq` ne change pas. Déclarer `MUTATION_FULL = "{{ get_env(name='MUTATION_FULL', default='') }}"` dans la partie commune (`[env]` de `mutation-go.toml`).
- Modify: `.config/mise.toml` : le même `[env]` et la même branche dans `mutation:go`.
- Create: `.github/workflows/mutation-full.yml` : `on: schedule` (hebdomadaire) et `workflow_dispatch` ; checkout complet ; mise ; `MUTATION_FULL=1 mise run mutation`.
- Modify: `claude/skills/repo-tooling/SKILL.md` (section « Mutation ») : le mode complet, le workflow planifié GitHub et l'équivalent GitLab (pipeline planifié, règle `$CI_PIPELINE_SOURCE == "schedule"`).
- La branche « complet / diff » est un séquencement (un `if` sur une variable) : bash autorisé par la règle des scripts.
