# Pas de code mort : la couverture des tests de frontière — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Do NOT use `stacked-tdd-mr` nor `harness gate` on this repository (spec `2026-11-13-filets-de-refactoring-design.md`, D9): the red and green of each task are seen by hand and quoted.

**Goal:** donner à la porte un mode `harness gate coverage` qui lance la tâche de test une fois, lit ses rapports de couverture et échoue pour tout code Go ou Python de production qu'aucun test de frontière n'exécute ; puis écrire dans les skills ce qu'un dépôt applicatif doit fournir pour produire ces rapports.

**Architecture:** un paquet de domaine pur, `harness/internal/domain/coverage`, lit les deux formats (profil Go, LCOV), les réunit, lit `COVERAGE.md` et rend des constats ; il n'importe que la bibliothèque standard (`go/parser` compris). Le mode vit dans `harness/internal/app/gate/coverage.go` : il passe par les ports existants (`Git`, `Files`, `Tasks`) pour lister les fichiers suivis, lancer la tâche et lire les rapports, puis imprime. Les textes (procédure, skills de stack, gabarits) viennent après le binaire, parce qu'ils citent une commande qui doit exister.

**Tech Stack:** Go (binaire `harness/`, tests boîte noire `harness/acceptance/`, gotestsum), mise (`.config/mise.toml` : `test:go`, `check`), `go build -cover` et `go tool covdata`, coverage.py ≥ 7.17, Docker Compose.

**Spec:** `claude/docs/specs/2026-11-16-couverture-de-frontiere-design.md` (ADR : `docs/adr/0002-le-code-mort-se-prouve-par-la-couverture-des-tests-de-frontiere.md`)

## Global Constraints

- Lignes de sortie, telles quelles : `COVERAGE <chemin>: <n> files`, `COVERAGE missing: <tâche> n'a pas écrit <chemin>`, `TASK FAILED <tâche>`, `UNCOVERED <fichier>:<début>-<fin>`, `UNCOVERED BRANCH <fichier>:<ligne>`, `NOT MEASURED <fichier>`, `PRAGMA <fichier>:<ligne>`, `ACCEPTED <fichier>:<début>-<fin>: <raison>`, `ACCEPTED <fichier>: <raison>`, `STALE EXCEPTION <fichier>: <texte>`, `ERROR RETURNS IGNORED <n>`, `NOT JUDGED <n> files: neither Go nor Python`. Dernière ligne `PASS` (code 0) ou `FAIL` (code 1) ; erreur d'usage ou d'environnement : `ERROR: <msg>`, `FAIL`, code 2.
- Ordre de sortie : la sortie de la tâche, les lignes `COVERAGE` dans l'ordre des options, les constats triés par fichier puis par ligne, les `STALE EXCEPTION`, `ERROR RETURNS IGNORED`, `NOT JUDGED`, puis `PASS` ou `FAIL`. Un bloc d'une ligne s'écrit `32-32`.
- La tâche de test est lancée une seule fois par appel, quel que soit le volume de code. Aucun contrôle ne relance la suite par bloc, par fichier ou par ligne.
- Aucune dépendance nouvelle dans `harness/go.mod` : le code Go se lit avec `go/parser` de la bibliothèque standard.
- Le mode ne lit ni `TESTS.md` ni slug, et n'exige pas que la tâche soit branchée.
- Noms fixés : option `--coverage` (répétable, obligatoire), option `--baseline`, fichier `COVERAGE.md` à la racine donnée par `--repo`, raison `dette initiale`, entrée `fichier entier`. Rapports d'un dépôt applicatif : `.tdd/cover/go.txt` (profil Go) et `.tdd/cover/py.lcov` (LCOV) ; données brutes sous `.tdd/cover/go/` et `.tdd/cover/py/`.
- TDD à la main : l'agent `acceptance-writer` écrit les tests du critère et la section de `claude/TESTS.md`, sans lire le code de production, et commite `test(<slug>): acceptance` ; l'orchestrateur lance les tests et cite la sortie rouge ; l'agent `implementer` écrit le code et commite `feat(<slug>): …` ; l'orchestrateur cite la sortie verte puis la dernière ligne de `mise run check`.
- Ce plan ne contient aucun code de test d'acceptation : chaque tâche donne les IDs, les noms des tests et la forme du dépôt d'essai ; le texte du critère est celui de la spec, recopié en entier dans `claude/TESTS.md`.
- Une tâche = une branche = une MR, empilée sur la précédente (la première sur `feat/couverture-plan`). CI verte avant de passer pour finie. Ne jamais fusionner.
- Commits conventionnels (`cog`), sans attribution d'IA : ni `Co-Authored-By`, ni ligne « Generated with », ni lien de session. Les sous-agents reçoivent la même règle et tournent en Sonnet au plus.
- Skills et agents en anglais, dans le style des skills actuels ; specs, `TESTS.md`, `COVERAGE.md` et gabarits de critères en français. Une règle s'énonce sans nommer un projet.
- Aucun critère ni test pour un texte. Une affirmation technique d'un skill n'est écrite qu'après avoir été vue sur une application d'essai, et porte « verified <date> » avec la version de l'outil.
- Applications d'essai hors du dépôt, dans le dossier de travail temporaire de la session (`<essais>/go-app`, `<essais>/py-app`, `<essais>/NOTES.md`). Rien n'est poussé sur une forge.
- Le dépôt du harnais lui-même n'est pas mis sous `gate coverage`.

## Review Focus

1. Un rapport qui nomme un fichier que git ne suit pas (LCOV sans réglage `source` : les paquets de `.venv` y figurent) : aucun constat pour lui, aucun `NOT MEASURED` inversé. → test unitaire `TestFindingsIgnoreUntrackedFiles` (tâche 2).
2. Les formes réelles d'une remontée d'erreur : `if err := f(); err != nil { return err }` et `else if err != nil { return err }` sont écartées ; un corps qui écrit dans un journal avant le `return`, et un `else`, sont jugés. → test unitaire `TestErrorReturnsForms` (tâche 4).
3. `--baseline` quand la tâche échoue ou qu'un rapport manque : `COVERAGE.md` n'est ni créé ni modifié. → test `TestCOV009_BaselineWritesNothingWhenTaskFails` (tâche 3).
4. Un texte d'exception qui contient un accent grave (chaîne brute Go) : écrit par `--baseline` puis relu, il accepte le même constat. → test unitaire `TestExceptionsRoundTripBackticks` (tâche 3).
5. Deux `go.mod` suivis dont l'un est dans un sous-dossier de l'autre : un fichier du profil revient au module dont le chemin est le plus long. → test unitaire `TestGoProfileNestedModules` (tâche 1).

---

## Partie A — le mode `harness gate coverage`

Forme commune des dépôts d'essai (aides à ajouter dans `harness/acceptance/helpers_test.go` par la tâche 1, réutilisées ensuite) :

- `covRepo(t, f files, reports files, rc int) *repo` : commite `f`, un `mise.toml` (`miseToml(miseOpts{})`), `.gitignore` (`.tdd`), et un `run_test.sh` qui écrit chaque rapport de `reports` (clé = chemin sous le dépôt, valeur = contenu, rangé dans un fichier suivi d'extension `.txt`) puis sort avec `rc`. Un rapport dont le contenu est vide n'est pas écrit par la tâche.
- `(r *repo) coverage(extra ...string) (int, string)` : `r.gate("coverage", "--task", "test", …)` avec une option `--coverage` par rapport de `covRepo`, dans l'ordre des clés triées, puis `extra`.
- Règle pour toutes les tâches : un dépôt d'essai ne suit aucun fichier `.go` ou `.py` de production autre que ceux que ses rapports mesurent, sauf quand le critère le demande. Ainsi la tâche 3 (`NOT MEASURED`) ne rend rouge aucun test antérieur.
- Les fichiers Go des dépôts d'essai sont du vrai code Go qui se lit avec `go/parser` (la tâche 4 les lira), avec un `go.mod` `module example.test/app`.

### Task 1: Un bloc Go jamais exécuté fait échouer la porte

**Slug:** `couverture-go` · **Branche:** `feat/couverture-go` (depuis `feat/couverture-plan`) · **Critères:** COV-001, COV-003, COV-007 · **Frontière:** `harness gate coverage`, sortie standard et code de sortie · **Commande:** `mise run test`

**Files:**
- Create: `harness/acceptance/couverture_test.go` ; Modify: `harness/acceptance/helpers_test.go` (`covRepo`, `coverage`), `claude/TESTS.md`
- Create: `harness/internal/domain/coverage/measure.go`, `goprofile.go` (et leurs tests unitaires)
- Create: `harness/internal/app/gate/coverage.go`
- Modify: `harness/internal/app/gate/gate.go` (`Options`, table `modes` de `Run`), `harness/cmd/harness/gate.go` (`gateUsage`, `parseGate`)

**Interfaces:**
- Consumes: `(*run).runMise(repo, task) (int, string, error)` (124 = délai dépassé), `r.g.git.Run`, `r.g.files.Read/Remove/Resolve`, `tdd.IsTestPath`, `failf`.
- Produces:
  - `gate.Options.Coverage []string` ; mode `"coverage"` → `func (r *run) coverage() (bool, error)`.
  - `coverage.Pos{Line, Col int}` ; `coverage.Measure` (ce que les rapports disent, par fichier du dépôt) ; `coverage.ParseGoProfile(text string, modules map[string]string) (Measure, error)` où `modules` associe un chemin de module à son dossier dans le dépôt ; `coverage.Merge(ms ...Measure) Measure` ; `coverage.Parse(text string, modules map[string]string) (Measure, error)` qui choisit le format d'après le contenu (première ligne `mode:` → profil Go) et rend une erreur pour un contenu inconnu ; `(m Measure) Files() []string`.
  - `coverage.Finding{Kind, File string; Start, End int; At Pos; Text, Reason string}` ; `(m Measure) Findings(source func(file string) string) []Finding`, triés par fichier puis ligne ; `Text` est la ligne `Start` du fichier, espaces retirés.

**Tests à écrire** (`couverture_test.go`) : `TestCOV001_UncoveredGoBlockFails`, `TestCOV001_AllCoveredPasses` (clauses 2 et 3), `TestCOV001_EmptyBlockIsNotJudged`, `TestCOV003_ReportsComplementEachOther`, `TestCOV007_ReportNotWrittenFails` (le test pose lui-même le rapport périmé avant l'appel), `TestCOV007_FailedTaskFails`.

- [ ] **Step 1: acceptance-writer** — aides, tests, section `## couverture-go — Un bloc Go jamais exécuté fait échouer la porte` de `claude/TESTS.md` (critères COV-001, COV-003, COV-007 recopiés de la spec ; ligne `Branche : `feat/couverture-go` · Commande : `mise run test``). Commit `test(couverture-go): acceptance`.
- [ ] **Step 2: Voir rouge** — `go test -C harness ./acceptance -run 'TestCOV00[137]_' -count=1`. Attendu : FAIL, code de sortie 2 du binaire (`unknown mode "coverage"`). Citer la sortie.
- [ ] **Step 3: implementer** — `parseGate` : cas `"coverage"` avec `fs.Var(list{&o.Coverage}, "coverage", …)` obligatoire ; le message « a mode is required » et `gateUsage` nomment le mode. `coverage()` : (1) supprime chaque rapport ; (2) `runMise` ; code non nul → `TASK FAILED <tâche>`, faux ; (3) lit chaque rapport : absent → `COVERAGE missing: …`, faux ; sinon `COVERAGE <chemin>: <n> files` ; (4) `git ls-files` (échec → `failf("%s is not a git repository", repo)`), modules lus dans les `go.mod` suivis (ligne `module`), le chemin de module le plus long l'emporte ; (5) réunit, garde les fichiers suivis de production (`.go`, hors `tdd.IsTestPath`), imprime `UNCOVERED`, puis `ERROR RETURNS IGNORED 0` (le compte devient réel à la tâche 4). Un bloc à 0 instruction n'entre pas dans la mesure. Tests unitaires du domaine, dont `TestGoProfileNestedModules`.
- [ ] **Step 4: Voir vert** — même commande : PASS ; `mise run check` vert. Citer la dernière ligne de chacun.
- [ ] **Step 5: Commit** `feat(couverture-go): harness gate coverage lit un profil Go`, pousser, MR sur `feat/couverture-plan`.

### Task 2: Une ligne ou une branche Python jamais exécutée fait échouer la porte

**Slug:** `couverture-python` · **Branche:** `feat/couverture-python` (depuis `feat/couverture-go`) · **Critères:** COV-002, COV-008 · **Frontière et commande:** comme la tâche 1

**Files:**
- Modify: `harness/acceptance/couverture_test.go`, `claude/TESTS.md`
- Create: `harness/internal/domain/coverage/lcov.go`, `pragma.go` (et tests unitaires)
- Modify: `harness/internal/domain/coverage/measure.go`, `harness/internal/app/gate/coverage.go`

**Interfaces:**
- Consumes: `coverage.Parse`, `Merge`, `Findings` (tâche 1).
- Produces: `coverage.ParseLCOV(text string) (Measure, error)` (chemins `SF:` relatifs à la racine ; compteur `-` d'une ligne `BRDA:` = jamais prise) ; `Parse` reconnaît LCOV à une ligne `SF:` ; `Findings` rend aussi les constats `UNCOVERED BRANCH` et groupe les lignes Python jamais exécutées qu'aucune ligne exécutée ne sépare dans le rapport ; `coverage.Pragmas(file, source string) []Finding` (`Kind` `PRAGMA`, pour `# pragma: no cover` et `# pragma: no branch`).

**Tests à écrire** : `TestCOV002_UncoveredPythonLinesAndBranch` (les quatre clauses), `TestCOV008_PragmaIsRefused` (les deux clauses).

- [ ] **Step 1: acceptance-writer** — tests et section `## couverture-python — Une ligne ou une branche Python jamais exécutée fait échouer la porte`. Commit `test(couverture-python): acceptance`.
- [ ] **Step 2: Voir rouge** — `go test -C harness ./acceptance -run 'TestCOV00[28]_' -count=1`. Attendu : FAIL (`ERROR:` format de rapport inconnu pour COV-002 ; `PASS` au lieu de `FAIL` pour COV-008). Citer.
- [ ] **Step 3: implementer** — LCOV, groupement, branches (une branche est prise dès qu'un rapport la dit prise ; pas de `UNCOVERED BRANCH` pour une ligne jamais exécutée) ; la porte lit chaque fichier `.py` suivi de production et imprime ses `PRAGMA`. Tests unitaires, dont `TestFindingsIgnoreUntrackedFiles`.
- [ ] **Step 4: Voir vert** — `go test -C harness ./acceptance -run 'TestCOV' -count=1` : PASS ; `mise run check` vert. Citer.
- [ ] **Step 5: Commit** `feat(couverture-python): harness gate coverage lit un rapport LCOV`, pousser, MR sur `feat/couverture-go`.

### Task 3: Exceptions écrites, fichiers non mesurés, dette de départ

**Slug:** `couverture-exceptions` · **Branche:** `feat/couverture-exceptions` (depuis `feat/couverture-python`) · **Critères:** COV-005, COV-006, COV-009 · **Frontière:** comme la tâche 1, plus le fichier `COVERAGE.md` du dépôt pour COV-009

**Files:**
- Modify: `harness/acceptance/couverture_test.go`, `claude/TESTS.md`
- Create: `harness/internal/domain/coverage/exceptions.go`, `gofile.go` (et tests unitaires)
- Modify: `harness/internal/app/gate/coverage.go`, `gate.go` (`Options.Baseline`), `ports.go` (`Files.Write`), `harness/internal/adapters/fsys/` (méthode `Write`), `harness/cmd/harness/gate.go` (`--baseline`, `gateUsage`)

**Interfaces:**
- Consumes: `coverage.Finding`, `Findings`, `Pragmas`, `(Measure).Files()`.
- Produces:
  - `coverage.ParseExceptions(md string) (Exceptions, error)` (sections `## <fichier>`, entrées `` - `<texte>` : <raison> `` et `- fichier entier : <raison>` ; raison vide ou entrée hors section → erreur, donc `ERROR:` et code 2) ; `(e Exceptions) Apply(fs []Finding) (accepted, refused []Finding, stale []Stale)` avec `Stale{File, Text string}` ; `PRAGMA` n'est jamais accepté ; `(e Exceptions) WithDebt(refused []Finding) Exceptions` ; `(e Exceptions) Markdown() string`, qui garde les entrées déjà là dans leur ordre.
  - `coverage.HasFunc(goSource string) bool` ; `Files.Write(path, content string) error` ; `gate.Options.Baseline bool`.
  - `NOT MEASURED` : `Finding` de `Kind` `NOT MEASURED`, `Start` 0, pour tout fichier `.go` ou `.py` suivi, de production, absent de la mesure, sauf un fichier Go pour lequel `HasFunc` est faux.

**Tests à écrire** : `TestCOV005_WrittenExceptionIsAccepted`, `TestCOV005_StaleExceptionFails`, `TestCOV006_UnmeasuredFileFails` (clauses 1 à 4), `TestCOV006_WholeFileExceptionIsAccepted`, `TestCOV009_BaselineWritesTheDebt` (les deux clauses ; le test lit `COVERAGE.md`), `TestCOV009_BaselineWritesNothingWhenTaskFails`.

- [ ] **Step 1: acceptance-writer** — tests et section `## couverture-exceptions — Une exception s'écrit, avec sa raison, hors du code`. Commit `test(couverture-exceptions): acceptance`.
- [ ] **Step 2: Voir rouge** — `go test -C harness ./acceptance -run 'TestCOV00[569]_' -count=1`. Attendu : FAIL (`UNCOVERED` au lieu de `ACCEPTED`, pas de `NOT MEASURED`, `--baseline` refusé en code 2). Citer.
- [ ] **Step 3: implementer** — lecture de `COVERAGE.md` (absent = aucune exception) ; texte entre accents graves selon la règle des portions de code Markdown (autant d'accents graves qu'il en faut, une espace de chaque côté si le texte commence ou finit par un accent grave) ; une entrée accepte toutes les occurrences de son texte dans le fichier ; `fichier entier` accepte `NOT MEASURED` et tous les constats du fichier ; `NOT JUDGED <n> files: neither Go nor Python` compte les fichiers suivis dont l'extension n'est ni `.go` ni `.py`. `--baseline` : écrit une entrée par texte distinct et par fichier, `fichier entier` pour `NOT MEASURED`, raison `dette initiale` ; ces constats ne font alors pas échouer ; rien n'est écrit après `TASK FAILED` ou `COVERAGE missing`. Tests unitaires, dont `TestExceptionsRoundTripBackticks`.
- [ ] **Step 4: Voir vert** — `go test -C harness ./acceptance -run 'TestCOV' -count=1` : PASS ; `mise run check` vert. Citer.
- [ ] **Step 5: Commit** `feat(couverture-exceptions): COVERAGE.md, fichiers non mesurés et --baseline`, pousser, MR sur `feat/couverture-python`.

### Task 4: Une remontée d'erreur Go est écartée et comptée

**Slug:** `couverture-remontees` · **Branche:** `feat/couverture-remontees` (depuis `feat/couverture-exceptions`) · **Critère:** COV-004 · **Frontière et commande:** comme la tâche 1

**Files:**
- Modify: `harness/acceptance/couverture_test.go`, `claude/TESTS.md`
- Create: `harness/internal/domain/coverage/goerr.go` (et tests unitaires)
- Modify: `harness/internal/app/gate/coverage.go`

**Interfaces:**
- Consumes: `coverage.Finding.At` (position de début du bloc dans le profil), `Exceptions.Apply`.
- Produces: `coverage.ErrorReturns(goSource string) (map[Pos]bool, error)` : la position de chaque `return` qui est la seule instruction du corps d'un `if` dont la condition est `<ident> != nil`, l'identifiant étant `err` ou finissant par `err` ou `Err`. La porte retire, avant `Apply`, tout constat `UNCOVERED` d'un fichier Go dont `At` est dans cet ensemble, et imprime leur nombre.

**Test à écrire** : `TestCOV004_ErrorReturnIsIgnoredAndCounted` (les deux clauses).

- [ ] **Step 1: acceptance-writer** — test et section `## couverture-remontees — Une remontée d'erreur Go est écartée et comptée`. Commit `test(couverture-remontees): acceptance`.
- [ ] **Step 2: Voir rouge** — `go test -C harness ./acceptance -run 'TestCOV004_' -count=1`. Attendu : FAIL (`UNCOVERED` pour le bloc `if err != nil`, `ERROR RETURNS IGNORED 0`). Citer.
- [ ] **Step 3: implementer** — `go/parser` et `go/ast` ; un fichier Go qui ne se lit pas → `failf`. Tests unitaires, dont `TestErrorReturnsForms`.
- [ ] **Step 4: Voir vert** — `go test -C harness ./acceptance -run 'TestCOV' -count=1` : PASS ; `mise run check` vert. Citer.
- [ ] **Step 5: Vérifier sur un vrai profil** — construire le binaire et le lancer sur une copie jetable du harnais avec le profil du 16/11 (une tâche `test` qui ne fait que copier ce profil) : la ligne `ERROR RETURNS IGNORED` doit donner 39 pour les paquets que ce profil mesure à plus de 0 %, à l'écart près expliqué dans la MR. Citer la ligne. Rien de cette copie n'est commité.
- [ ] **Step 6: Commit** `feat(couverture-remontees): les remontées d'erreur Go sont écartées et comptées`, pousser, MR sur `feat/couverture-exceptions`.

---

## Partie B — les textes

### Task 5: La procédure demande la couverture

**Branche:** `docs/couverture-procedure` (depuis `feat/couverture-remontees`) · **Critères:** aucun (texte)

**Files:**
- Modify: `claude/skills/stacked-tdd-mr/SKILL.md`, `claude/agents/implementer.md`

- [ ] **Step 1: `stacked-tdd-mr`** — une étape après la porte verte de chaque tâche et une en fin de pile, avant la revue finale : quand `repo-stack` nomme un skill de stack, `~/.local/bin/harness gate coverage --task test --repo <worktree> --coverage .tdd/cover/go.txt` et/ou `--coverage .tdd/cover/py.lcov` ; citer le code de sortie et la dernière ligne. Sens de chaque ligne de sortie (liste des Global Constraints). `UNCOVERED`, `UNCOVERED BRANCH`, `NOT MEASURED` : constat pour l'implementer, qui retire le code ou répond `UNTESTED` ; l'orchestrateur tranche comme pour `UNTESTED` (un critère de plus, ou une entrée de `COVERAGE.md` après un `Ruling:` au registre). `STALE EXCEPTION` : l'orchestrateur retire l'entrée. `PRAGMA` : l'implementer retire le commentaire. Seul l'orchestrateur écrit `COVERAGE.md`. `--baseline` : une fois, à l'adoption d'un dépôt existant, après un `Ruling:`. Un dépôt sans skill de stack : l'étape ne s'applique pas.
- [ ] **Step 2: `implementer.md`** — tout bloc de production doit être atteint par un test d'acceptation ; ne modifie jamais `COVERAGE.md` ; n'écrit pas `# pragma: no cover` ni `# pragma: no branch` ; un chemin qu'aucun critère ne demande se signale par `UNTESTED`.
- [ ] **Step 3: Vérifier** — `grep -n 'gate coverage' claude/skills/stacked-tdd-mr/SKILL.md` ; chaque option citée existe dans `harness gate --help` du binaire construit sur cette branche (citer les deux sorties) ; `mise run check` vert.
- [ ] **Step 4: Commit** `docs(stacked-tdd-mr): la couverture se juge après la porte verte`, pousser, MR sur `feat/couverture-remontees`.

### Task 6: Ce qu'un dépôt applicatif fournit

**Branche:** `docs/couverture-applications` (depuis `docs/couverture-procedure`) · **Critères:** aucun (texte vérifié sur deux applications d'essai)

**Files:**
- Modify: `claude/skills/compose-dev/SKILL.md`, `claude/skills/stack-go-app/SKILL.md`, `claude/skills/stack-py-app/SKILL.md`, `claude/skills/repo-tooling/SKILL.md`
- Modify: `claude/skills/repo-tooling/templates/mise.toml`, `compose.go.yaml`, `compose.py.yaml`, `gitignore`

- [ ] **Step 1: Deux applications d'essai** sous `<essais>/` : `go-app` et `py-app`, chacune avec un service HTTP en conteneur (deux routes) et un outil en ligne de commande, construites d'après les skills de stack actuels, avec un test de frontière qui n'appelle qu'une des deux routes.
- [ ] **Step 2: Faire sortir les rapports**, et noter dans `<essais>/NOTES.md` chaque réglage et la commande qui le prouve :
  - Go : argument de construction du Dockerfile de production, vide par défaut, qui ajoute `-cover -coverpkg=./...` ; `GOCOVERDIR` sur un dossier monté ; binaire de l'outil construit de même par la tâche de test ; `go tool covdata textfmt -i=.tdd/cover/go -o=.tdd/cover/go.txt` en fin de tâche `test`.
  - Python : coverage.py installé dans l'image par le même genre d'argument, démarré par `COVERAGE_PROCESS_START` ; `branch`, `parallel`, `relative_files`, `sigterm`, `source` ; `COVERAGE_FILE` sur le dossier monté ; `init: true` dans compose ; `coverage combine` puis `coverage lcov -o .tdd/cover/py.lcov`.
  - Les deux : dossier monté inscriptible par l'utilisateur du conteneur ; arrêt par SIGTERM avant la conversion ; `pytest` et `go test` non instrumentés.
- [ ] **Step 3: Passer la porte** — sur chaque application, `harness gate coverage --task test …` : `UNCOVERED` sur la route jamais appelée et `FAIL` ; après ajout de l'appel manquant, `PASS`. Vérifier aussi que l'image construite sans l'argument ne contient pas la mesure (pas d'avertissement `GOCOVERDIR`, `coverage` absent de l'image Python). Citer les six sorties dans `NOTES.md`.
- [ ] **Step 4: Écrire les textes** d'après `NOTES.md` seulement : `compose-dev` (l'exception à la ligne « same target as the published image » : même Dockerfile, même cible, un argument de construction que seules les tâches de test passent ; `mise run dev` et l'image publiée ne l'ont pas) ; `stack-go-app` et `stack-py-app` (la mesure tourne dans le produit, jamais dans le lanceur de tests ; chemins des rapports ; « en Go la mesure porte sur des blocs, en Python sur des lignes et des branches ») ; `repo-tooling` et ses gabarits (tâche `test` qui lance toutes les suites puis convertit ; `.tdd/` ignoré). Chaque affirmation porte « verified <date> » et la version de l'outil. Ce qui n'a pas été essayé (serveur Python à plusieurs processus, image distroless, macOS) est écrit comme non vérifié.
- [ ] **Step 5: Vérifier** — `mise run check` vert ; relire chaque phrase ajoutée contre `NOTES.md`.
- [ ] **Step 6: Commit** `docs(stack-skills): un dépôt applicatif produit ses rapports de couverture`, pousser, MR sur `docs/couverture-procedure`.

---

## Fin de pile

- Revue finale de la pile entière (lecture seule), puis rapport d'état : MR relues par `gh`, décisions prises en route, ce qui n'a pas été vérifié.
- Fusion seulement à la demande de l'utilisateur : `harness merge-stack <MR du haut>`.
