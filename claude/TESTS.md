# Tests

## forme — Forme des preuves du harnais
Branche : `feat/forme` · Commande : `mise run test`

### FRM-001 — Le critère est écrit en toutes lettres dans l'entrée
- **Critère** :
  - **Étant donné** un `TESTS.md` dont l'entrée `X-001` du slug jugé ne porte qu'un renvoi (`**Critère** : docs/specs/x.md › X-001`), et dont une section d'un autre slug est à l'ancien format
  - **Quand** `harness gate red`, `green` ou `characterize` est lancé sur le slug
  - **Alors** la sortie contient `INCOMPLETE X-001` et se termine par `FAIL`, sans aucune ligne `INCOMPLETE` pour l'autre section
- **Test** : `../harness/acceptance/forme_test.go` › `TestFRM001_RedIncompleteEntry` ; `../harness/acceptance/forme_test.go` › `TestFRM001_GreenIncompleteEntry` ; `../harness/acceptance/forme_test.go` › `TestFRM001_CharacterizeIncompleteEntry`

### FRM-003 — Un rapport absent dit quoi faire
- **Critère** :
  - **Étant donné** une tâche de test qui sort à 0 sans écrire de rapport
  - **Quand** `harness gate characterize` est lancé
  - **Alors** la sortie contient une ligne `REPORT missing` qui nomme la tâche et le chemin `.tdd/junit.xml`, et se termine par `FAIL`
- **Test** : `../harness/acceptance/forme_test.go` › `TestFRM003_CharacterizeReportMissing`

### FRM-005 — La spec jugée est celle qui est commitée
- **Critère** :
  - **Étant donné** une spec située dans le dépôt, avec une modification non commitée
  - **Quand** `harness gate red` ou `characterize` est lancé avec `--spec`
  - **Alors** la sortie contient `SPEC NOT COMMITTED` et se termine par `FAIL`
- **Test** : `../harness/acceptance/forme_test.go` › `TestFRM005_RedSpecNotCommitted` ; `../harness/acceptance/forme_test.go` › `TestFRM005_CharacterizeSpecNotCommitted`

## branchement — Branchement de la tâche de test
Branche : `feat/branchement` · Commande : `mise run test`

### FRM-004 — La tâche de test tourne quelque part
- **Critère** :
  - **Étant donné** un slug dont la `Commande` est `mise run test:render`, tâche citée par aucun hook de commit ni job CI du dépôt, ni dépendance d'une tâche qui l'est
  - **Quand** `harness gate characterize` ou `green` est lancé
  - **Alors**
    - la sortie contient `NOT WIRED test:render` et se termine par `FAIL`
    - une fois la tâche citée dans `.pre-commit-config.yaml` ou `.gitlab-ci.yml`, la ligne `NOT WIRED` n'apparaît plus
    - quand le hook ne cite que `mise run check` et que `test:render` est une dépendance (`depends`) de `check` dans `mise.toml`, la ligne `NOT WIRED` n'apparaît pas
- **Test** : `../harness/acceptance/branchement_test.go` › `TestFRM004_GreenTaskNotWired` ; `../harness/acceptance/branchement_test.go` › `TestFRM004_CharacterizeTaskNotWired` ; `../harness/acceptance/branchement_test.go` › `TestFRM004_GreenTaskWired` ; `../harness/acceptance/branchement_test.go` › `TestFRM004_CharacterizeTaskWired` ; `../harness/acceptance/branchement_test.go` › `TestFRM004_GreenTaskWiredByDependency`

## mutant-mise — Mutant du code écrit dans une tâche mise
Branche : `feat/mutant-mise` · Commande : `mise run test`

### PRV-008 — Un mutant du code écrit dans une tâche mise est jugé
- **Critère** :
  - **Étant donné** un dépôt dont le code testé est le corps de la tâche `[tasks."bootstrap:agent"]` de `mise.toml`, jugé par la tâche `test:bootstrap`
  - **Quand** `harness gate characterize` reçoit un mutant qui modifie le corps de `bootstrap:agent`, puis un mutant qui modifie le corps de `test:bootstrap`
  - **Alors**
    - le premier est appliqué et jugé (`KILLED` ou `SURVIVED`)
    - le second est refusé par `MUTANT INVALID … touches test task test:bootstrap`
    - un mutant d'un autre fichier que `mise.toml`, dont le hunk porte sur des numéros de ligne qui sont, dans `mise.toml`, ceux du corps de `test:bootstrap`, est appliqué et jugé (`KILLED` ou `SURVIVED`)
- **Test** : `../harness/acceptance/mutant_mise_test.go` › `TestPRV008_MutantOfCodeTaskIsJudged` ; `../harness/acceptance/mutant_mise_test.go` › `TestPRV008_MutantOfTestTaskIsRefused` ; `../harness/acceptance/mutant_mise_test.go` › `TestPRV008_MutantOfOtherFileIsJudged`

## preuve-sensible — Un mutant par clause du « Alors »
Branche : `feat/preuve-sensible` · Commande : `mise run test`

### PRV-001 — Un mutant par clause est exigé
- **Critère** :
  - **Étant donné** un slug dont le critère `X-001` a un « Alors » à deux clauses, et un dossier de preuves qui contient `X-001.patch` et `X-001.neutral.patch` mais pas `X-001.2.patch`
  - **Quand** `harness gate characterize` est lancé sur le commit d'acceptation
  - **Alors** la sortie contient `MUTANT MISSING X-001 clause 2` et se termine par `FAIL`
- **Test** : `../harness/acceptance/clauses_test.go` › `TestPRV001_MissingMutantForClause2`

### PRV-002 — Une clause qu'aucun test ne défend est refusée
- **Critère** :
  - **Étant donné** le même critère à deux clauses, les deux mutants présents, et un test qui n'échoue que sous le premier
  - **Quand** `harness gate characterize` est lancé
  - **Alors** la sortie contient `KILLED X-001 clause 1` et `SURVIVED X-001 clause 2`, et se termine par `FAIL`
- **Test** : `../harness/acceptance/clauses_test.go` › `TestPRV002_ClauseWithoutDefendingTestSurvives`

## preuve-specifique — Les preuves de spécificité (changement neutre)
Branche : `feat/preuve-specifique` · Commande : `mise run test`

### PRV-003 — Un changement neutre par critère est exigé
- **Critère** :
  - **Étant donné** un critère `X-001` dont le mutant est présent et tué, sans `X-001.neutral.patch`
  - **Quand** `harness gate characterize` est lancé
  - **Alors**
    - la sortie contient `NEUTRAL MISSING X-001` et se termine par `FAIL`
    - relancée avec un `X-001.neutral.patch` qui ne s'applique pas au dépôt, elle contient `NEUTRAL INVALID X-001` et se termine par `FAIL`
- **Test** : `../harness/acceptance/clauses_test.go` › `TestPRV003_MissingNeutralChange` ; `../harness/acceptance/clauses_test.go` › `TestPRV003_NeutralChangeNotApplyingIsInvalid`

### PRV-004 — Un test qui casse sous un changement neutre est refusé
- **Critère** :
  - **Étant donné** un critère `X-001` dont le test affirme le nom d'un élément du catalogue, et un changement neutre qui renomme cet élément
  - **Quand** `harness gate characterize` est lancé
  - **Alors** la sortie contient `BRITTLE X-001` avec le nom du test en échec, et se termine par `FAIL`
- **Test** : `../harness/acceptance/clauses_test.go` › `TestPRV004_TestBreakingUnderNeutralChangeIsBrittle`

### PRV-005 — Un test sensible et spécifique passe
- **Critère** :
  - **Étant donné** un critère `X-001` dont le « Alors » a deux clauses en sous-liste, suivie de la ligne `- **Test**`, ses deux mutants, son changement neutre, et des tests qui échouent sous chaque mutant et passent sous le changement neutre
  - **Quand** `harness gate characterize` est lancé
  - **Alors** la sortie contient `KILLED X-001 clause 1`, `KILLED X-001 clause 2` et `NEUTRAL OK X-001`, aucune ligne `MUTANT MISSING`, et se termine par `PASS`
- **Test** : `../harness/acceptance/clauses_test.go` › `TestPRV005_SensitiveAndSpecificTestPasses`

## preuve-necessaire — Un critère couvert par un autre est refusé
Branche : `feat/preuve-necessaire` · Commande : `mise run test`

### PRV-006 — Un critère couvert par un autre est refusé
- **Critère** :
  - **Étant donné** deux critères `X-001` et `X-002` jugés par la même tâche, où le mutant de `X-002` fait aussi échouer le test de `X-001`, et où le mutant de `X-001` ne fait pas échouer le test de `X-002`
  - **Quand** `harness gate characterize` est lancé
  - **Alors** la sortie contient `REDUNDANT X-002` en nommant `X-001`, ne contient pas `REDUNDANT X-001`, et se termine par `FAIL`
- **Test** : `../harness/acceptance/necessaire_test.go` › `TestPRV006_CriterionCoveredByAnotherIsRedundant`

## preuves-nouveau — Les preuves en mode nouveau
Branche : `feat/preuves-nouveau` · Commande : `mise run test`

### PRV-007 — Les preuves valent aussi pour une fonctionnalité nouvelle
- **Critère** :
  - **Étant donné** un commit d'acceptation et un commit `feat` qui rend ses tests verts
  - **Quand** `harness gate green` est lancé sans `--proofs` ni `--proof-exempt`
  - **Alors**
    - la sortie contient `PROOF MISSING` et se termine par `FAIL`
    - relancé avec `--proofs` et un test qui survit à son mutant, elle contient `SURVIVED` et se termine par `FAIL`
    - relancé avec `--proofs`, un test qui échoue sous son mutant et passe sous son changement neutre, elle contient `KILLED` et `NEUTRAL OK` pour le critère, aucune ligne `SURVIVED`, et se termine par `PASS`
    - relancé avec `--proof-exempt`, aucun mutant n'est appliqué et elle se termine par `PASS`
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV007_MissingProofsOptionFails` ; `../harness/acceptance/preuves_green_test.go` › `TestPRV007_SurvivingMutantFails` ; `../harness/acceptance/preuves_green_test.go` › `TestPRV007_KilledMutantAndNeutralChangePass` ; `../harness/acceptance/preuves_green_test.go` › `TestPRV007_ProofExemptAppliesNoMutant`

## neutre-sans-resultat — Un changement neutre sans résultat est invalide
Branche : `feat/neutre-sans-resultat` · Commande : `mise run test`

### PRV-010 — Un changement neutre sans résultat est invalide
- **Critère** :
  - **Étant donné** un critère `X-001` dont le mutant est tué, et un `X-001.neutral.patch` qui s'applique mais sous lequel la tâche de test ne produit aucun résultat pour les tests du critère
  - **Quand** `harness gate characterize` est lancé
  - **Alors** la sortie contient `NEUTRAL INVALID X-001`, aucune ligne `NEUTRAL OK X-001`, et se termine par `FAIL`
- **Test** : `../harness/acceptance/clauses_test.go` › `TestPRV010_NeutralChangeWithoutResultIsInvalid`

## test-garde — Un test gardé ne change pas
Branche : `feat/test-garde` · Commande : `mise run test`

### FRM-002 — Un test gardé ne change pas
- **Critère** :
  - **Étant donné** un fichier de test cité par un slug remplacé (`--replaces`), où le test `X-001`, présent dans `TESTS.md` avant et après, a une ligne modifiée, et où `X-002` est modifié et listé dans `--ac-changed`
  - **Quand** `harness gate characterize` est lancé
  - **Alors**
    - la sortie contient `KEPT TEST MODIFIED` avec le fichier et `X-001`, aucune ligne de ce type pour `X-002`, et se termine par `FAIL`
    - quand `X-001` est laissé intact et qu'un test de `X-002` est ajouté à la fin du fichier, après une ligne vide, la sortie ne contient aucune ligne `KEPT TEST MODIFIED`
    - quand `X-001` est porté par deux tests du fichier, tous deux laissés intacts, la sortie ne contient aucune ligne `KEPT TEST MODIFIED`
- **Test** : `../harness/acceptance/tests_gardes_test.go` › `TestFRM002_KeptTestModifiedIsReported` ; `../harness/acceptance/tests_gardes_test.go` › `TestFRM002_TestAppendedAfterKeptTestIsNotReported` ; `../harness/acceptance/tests_gardes_test.go` › `TestFRM002_KeptIdentifierWithTwoTestsIsNotReported`

## rejeu — Rejeu des mutants sur la base et la tête d'une pile
Branche : `feat/rejeu` · Commande : `mise run test`

### RPL-001 — Une protection perdue par la pile est signalée
- **Critère** :
  - **Étant donné** deux commits d'un dépôt, un mutant qui fait échouer la tâche sur le premier et la laisse passer sur le second
  - **Quand** `harness gate replay --base <premier> --head <second>` est lancé
  - **Alors**
    - la sortie contient `LOST` avec le nom du mutant et se termine par `FAIL`
    - relancé avec `--accept-lost <nom>`, elle contient `LOST ACCEPTED` et se termine par `PASS`
    - relancé en échangeant `--base` et `--head`, elle contient `GAINED` avec le nom du mutant et se termine par `PASS`
- **Test** : `../harness/acceptance/rejeu_test.go` › `TestRPL001_LostProtectionIsReportedThenAccepted` ; `../harness/acceptance/rejeu_test.go` › `TestRPL001_GainedProtectionIsReported`

### RPL-002 — Un rejeu sans perte passe et ne laisse rien derrière lui
- **Critère** :
  - **Étant donné** deux commits et deux mutants, l'un tué des deux côtés, l'autre vivant des deux côtés
  - **Quand** `harness gate replay` est lancé
  - **Alors**
    - la sortie contient `KEPT` pour le premier et `NEVER` pour le second, et se termine par `PASS`
    - après l'exécution, `git worktree list` et `git status --short` du dépôt sont ceux d'avant
- **Test** : `../harness/acceptance/rejeu_test.go` › `TestRPL002_LosslessReplayPassesAndLeavesNothing`

## clauses — L'entrée a autant de clauses que la spec
Branche : `feat/clauses` · Commande : `mise run test`

### FRM-006 — L'entrée a autant de clauses que la spec
- **Critère** :
  - **Étant donné** une spec dont le critère `X-001` a un « Alors » en liste de deux clauses, et un `TESTS.md` dont l'entrée `X-001` du slug jugé porte ce « Alors » sur une seule ligne
  - **Quand** `harness gate red` ou `characterize` est lancé avec `--spec`
  - **Alors** la sortie contient `CLAUSES X-001: 1 in TESTS.md, 2 in spec` et se termine par `FAIL`
- **Test** : `../harness/acceptance/clauses_test.go` › `TestFRM006_RedEntryWithFewerClausesThanSpecFails` ; `../harness/acceptance/clauses_test.go` › `TestFRM006_CharacterizeEntryWithFewerClausesThanSpecFails`

## revue-finale — Corrections de la revue finale
Branche : `feat/revue-finale` · Commande : `mise run test`

### FRM-007 — Une tâche atteinte par un motif de dépendance est branchée
- **Critère** :
  - **Étant donné** un slug dont la `Commande` est `mise run test:render`, un hook de commit qui lance `mise run test`, et un `mise.toml` où `[tasks.test]` a `depends = ["test:*"]`
  - **Quand** `harness gate green` ou `characterize` est lancé
  - **Alors** la sortie ne contient aucune ligne `NOT WIRED`
- **Test** : `../harness/acceptance/branchement_test.go` › `TestFRM007_GreenTaskReachedByDependencyPattern` ; `../harness/acceptance/branchement_test.go` › `TestFRM007_CharacterizeTaskReachedByDependencyPattern`

### PRV-011 — Un mutant sans en-tête `diff --git` est jugé
- **Critère** :
  - **Étant donné** un dossier de preuves dont le mutant de `X-001` est un patch qui commence par `--- a/<fichier>`, sans ligne `diff --git`, et qui casse le test du critère
  - **Quand** `harness gate characterize` est lancé
  - **Alors** la sortie contient `KILLED X-001`
- **Test** : `../harness/acceptance/clauses_test.go` › `TestPRV011_MutantWithoutDiffGitHeaderIsJudged`

### PRV-012 — Les preuves ne s'appliquent pas sur un arbre modifié
- **Critère** :
  - **Étant donné** un dépôt dont un fichier suivi porte une modification non commitée
  - **Quand** `harness gate green --proofs DIR` est lancé
  - **Alors** la sortie contient `TREE NOT CLEAN`, aucune ligne `MUTANT`, se termine par `FAIL`, et la modification non commitée est toujours dans le fichier
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV012_ProofsNotAppliedOnModifiedTree`

### RPL-003 — Une tâche qui échoue sans mutant arrête le rejeu
- **Critère** :
  - **Étant donné** deux commits dont le premier fait échouer la tâche sans aucun mutant
  - **Quand** `harness gate replay --base <premier> --head <second>` est lancé
  - **Alors**
    - la sortie contient `BASELINE NOT GREEN base`, aucune ligne `KEPT`, et se termine par `FAIL`
    - relancé en échangeant `--base` et `--head`, elle contient `BASELINE NOT GREEN head`, aucune ligne `KEPT`, et se termine par `FAIL`
- **Test** : `../harness/acceptance/rejeu_test.go` › `TestRPL003_FailingTaskWithoutMutantStopsTheReplay` ; `../harness/acceptance/rejeu_test.go` › `TestRPL003_SwappedBaseAndHeadNamesHead`

## test-garde-voisin — Un test gardé n'est pas dit modifié quand son voisin change
Branche : `feat/test-garde-voisin` · Commande : `mise run test`

### PRV-015 — Un test gardé n'est pas dit modifié quand son voisin change
- **Critère** :
  - **Étant donné** un fichier de test où le test du critère `X-001` est suivi d'un autre test, et un commit d'acceptation qui laisse le test de `X-001` intact au caractère près
  - **Quand** `harness gate characterize` est lancé sans `--ac-changed`
  - **Alors**
    - quand le test qui suit est un test pytest précédé d'un décorateur (ligne `@…` au-dessus de son nom) et que le commit le retire, si bien que le test suivant, précédé d'un autre décorateur, prend sa place, la sortie ne contient aucune ligne `KEPT TEST MODIFIED` qui nomme `X-001`
    - quand le fichier est un fichier helm-unittest à plusieurs documents et que le commit retire le document qui suit le test de `X-001` (séparateur `---` puis en-tête `suite:`), la sortie ne contient aucune ligne `KEPT TEST MODIFIED` qui nomme `X-001`
- **Test** : `../harness/acceptance/tests_gardes_test.go` › `TestPRV015_PytestDecoratedNeighbourRemovedDoesNotModifyKeptTest` ; `../harness/acceptance/tests_gardes_test.go` › `TestPRV015_HelmUnittestFollowingDocumentRemovedDoesNotModifyKeptTest`

## test-garde-dernier — Un test gardé devenu le dernier du fichier
Branche : `feat/test-garde-dernier` · Commande : `mise run test`

### PRV-016 — Un test gardé devenu le dernier du fichier n'est pas dit modifié
- **Critère** :
  - **Étant donné** un fichier de test pytest qui se termine par un bloc `if __name__ == "__main__":`, où le test du critère `X-001` est suivi d'un dernier test, et un commit d'acceptation qui retire ce dernier test en laissant le test de `X-001` intact au caractère près
  - **Quand** `harness gate characterize` est lancé sans `--ac-changed`
  - **Alors** la sortie ne contient aucune ligne `KEPT TEST MODIFIED` qui nomme `X-001`
- **Test** : `../harness/acceptance/tests_gardes_test.go` › `TestPRV016_KeptTestBecomingLastOfFileIsNotModified`

## fichiers-locaux — Fichiers locaux du harnais
Branche : `feat/fichiers-locaux` · Commande : `mise run test`

### LOC-001 — Un hôte déclaré en local donne son outil de forge
- **Critère** :
  - **Étant donné** un dépôt git dont `origin` vaut `https://forge.exemple.test/groupe/depot.git`, et un dossier local (variable `CLAUDE_LOCAL_DIR`) dont le fichier `forges` contient la ligne `forge.exemple.test glab`
  - **Quand** `harness local forge --repo <dépôt>` est lancé
  - **Alors** la sortie est `glab` et le code de sortie est 0
- **Test** : `../harness/acceptance/local_test.go` › `TestLOC001_DeclaredHostGivesItsForgeTool`

### LOC-002 — Sans fichier local, seuls les hôtes publics sont connus
- **Critère** :
  - **Étant donné** un dossier local vide (variable `CLAUDE_LOCAL_DIR`)
  - **Quand** `harness local forge --repo <dépôt>` est lancé
  - **Alors**
    - pour un dépôt dont `origin` vaut `git@github.com:groupe/depot.git`, la sortie est `gh` et le code de sortie est 0
    - pour un dépôt dont `origin` vaut `https://gitlab.com/groupe/depot.git`, la sortie est `glab` et le code de sortie est 0
    - pour un dépôt dont `origin` vaut `https://forge.exemple.test/groupe/depot.git`, le code de sortie est 1 et la sortie contient `hôte forge.exemple.test inconnu`
- **Test** : `../harness/acceptance/local_test.go` › `TestLOC002_GithubHostIsGh` ; `../harness/acceptance/local_test.go` › `TestLOC002_GitlabHostIsGlab` ; `../harness/acceptance/local_test.go` › `TestLOC002_UnknownHostExits1AndIsNamed`

### LOC-003 — Un cas réel local affine une règle sans être versionné
- **Critère** :
  - **Étant donné** un dossier local (variable `CLAUDE_LOCAL_DIR`) dont le fichier `cas-reels.md` contient une ligne `| PRV-015 | dépôt interne, critère X-103 |`
  - **Quand** `harness local cas-reels PRV-015` est lancé
  - **Alors**
    - la sortie contient la ligne du fichier qui nomme `PRV-015`, et le code de sortie est 0
    - quand le fichier `cas-reels.md` n'existe pas, la sortie est vide et le code de sortie est 0
- **Test** : `../harness/acceptance/local_test.go` › `TestLOC003_RealCaseLineNamingTheIDIsPrinted` ; `../harness/acceptance/local_test.go` › `TestLOC003_MissingFileGivesEmptyOutput`

## accept-lost-multiple — Plusieurs pertes acceptées se nomment chacune
Branche : `feat/harness-accept-lost-multiple` · Commande : `mise run test`

### RPL-004 — Plusieurs pertes acceptées se nomment chacune
- **Critère** :
  - **Étant donné** deux commits d'un dépôt et deux mutants, chacun fait échouer la tâche sur le premier et la laisse passer sur le second
  - **Quand** `harness gate replay --base <premier> --head <second>` est lancé avec `--accept-lost <mutant 1> --accept-lost <mutant 2>`
  - **Alors**
    - la sortie contient `LOST ACCEPTED` avec le nom de chacun des deux mutants et se termine par `PASS`
    - relancé avec le seul `--accept-lost <mutant 1>`, elle contient `LOST` avec le nom du mutant 2 et se termine par `FAIL`
- **Test** : `../harness/acceptance/rejeu_test.go` › `TestRPL004_EachAcceptedLossIsNamed`

## test-bruno — Un test Bruno porte son ID
Branche : `feat/harness-outils-de-test` · Commande : `mise run test`

### FRM-008 — Un test Bruno porte son ID
- **Critère** :
  - **Étant donné** un `TESTS.md` dont l'entrée `X-001` du slug jugé cite un fichier Bruno (`tests/api/x.bru`), ce fichier contenant `test("[X-001] <quoi>", function() {`, et un rapport JUnit dont le cas `[X-001] <quoi>` échoue
  - **Quand** `harness gate red` est lancé sur le slug
  - **Alors** la sortie contient `RED X-001`, aucune ligne `MISSING X-001`, et se termine par `PASS`
- **Test** : `../harness/acceptance/tests_gardes_test.go` › `TestFRM008_BrunoTestCarriesItsID`
## preuves-optionnelles — Le changement neutre se saute sur demande
Branche : `feat/harness-preuves-optionnelles` · Commande : `mise run test`

### PRV-017 — Le changement neutre se saute sur demande
- **Critère** :
  - **Étant donné** un commit d'acceptation, un commit `feat` vert, et un dossier de preuves qui contient un mutant tuant le test du critère et aucun changement neutre
  - **Quand** `harness gate green --proofs` est lancé avec `--no-neutral`
  - **Alors**
    - la sortie contient `KILLED` pour le critère et `NEUTRAL SKIPPED`, aucune ligne `NEUTRAL MISSING`, et se termine par `PASS`
    - relancé sans `--no-neutral`, elle contient `NEUTRAL MISSING` pour le critère et se termine par `FAIL`
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV017_NeutralChangeIsSkippedOnRequest`

## mutants-paralleles — Les mutants se jugent en parallèle sur demande
Branche : `feat/harness-mutants-paralleles` · Commande : `mise run test`

### PRV-019 — Les mutants se jugent en parallèle sur demande
- **Critère** :
  - **Étant donné** un critère à deux clauses, un mutant par clause, et une tâche de test qui, sous mutant, n'aboutit que si une autre exécution est en cours au même moment
  - **Quand** `harness gate characterize --proofs` est lancé avec `--jobs 2`
  - **Alors**
    - la sortie contient `KILLED` pour chacune des deux clauses et se termine par `PASS`
    - les deux exécutions sous mutant ont eu lieu dans deux dossiers distincts, hors du dépôt
    - après l'exécution, `git worktree list` et `git status --short` du dépôt sont ceux d'avant
- **Test** : `agents/bin/tests/test_acceptance_mutants_paralleles.py` › `test_PRV019_mutants_are_judged_in_parallel`

## recherche-semble — Une recherche d'exploration reçoit un rappel de Semble
Branche : `feat/semble` · Commande : `mise run test`

### RCH-001 — Une recherche qui se lit comme une question passe avec un rappel de Semble
- **Critère** :
  - **Étant donné** un appel `Grep` de motif `where are tokens refreshed`, un appel Bash `grep -rn "how is the stack merged" harness` et un appel Bash `rtk proxy rg "auth|login|session" src`
  - **Quand** le hook `search-hint` juge chacun
  - **Alors**
    - la réponse porte un `additionalContext` qui nomme `semble search`
    - elle ne porte aucune `permissionDecision`
- **Test** : `hooks/tests/search-hint.bats` › `RCH-001 a search that reads like a question passes with a reminder of semble`

### RCH-002 — Une recherche exacte passe en silence
- **Critère** :
  - **Étant donné** un appel `Grep` de motif `TF_PLUGIN_CACHE_DIR`, et les appels Bash `grep -rn "func NewMergeStack" harness`, `rg -A 3 "permissionDecision.*ask" claude` et `git log --oneline -5`
  - **Quand** le hook `search-hint` juge chacun
  - **Alors** il n'écrit rien et sort à 0
- **Test** : `hooks/tests/search-hint.bats` › `RCH-002 an exact search passes in silence`
## fusion-de-pile — Une pile de merge requests se fusionne et se prouve
Branche : `feat/fusion-de-pile` · Commande : `mise run test`

### FUS-001 — Une pile se fusionne du bas vers le haut et la branche par défaut porte le code du haut
- **Critère** :
  - **Étant donné** une pile de trois MR ouvertes `a` ← `b` ← `c`, la première sur `main`, toutes au pipeline vert
  - **Quand** `harness merge-stack 3` est lancé dans le dépôt
  - **Alors**
    - les trois MR sont fusionnées
    - l'arbre de `main` sur le dépôt distant est celui du haut de la pile avant la fusion
    - la sortie contient `STACK 3: #1 a -> #2 b -> #3 c`, `TREE IDENTICAL` et `CHECKS PASS main`, se termine par `PASS`, et le code de sortie est 0
- **Test** : `hooks/tests/merge-stack.bats` › `FUS-001 a stack is merged bottom first and main holds the code of its top`

### FUS-002 — Une MR seule se fusionne sans celles du dessus
- **Critère** :
  - **Étant donné** la même pile
  - **Quand** `harness merge-stack a` est lancé (la MR du bas, désignée par sa branche)
  - **Alors**
    - seule la MR du bas est fusionnée, les deux autres restent ouvertes
    - l'arbre de `main` est celui de la branche `a`, et la sortie se termine par `PASS`
- **Test** : `hooks/tests/merge-stack.bats` › `FUS-002 a single merge request is merged alone`

### FUS-003 — Rien n'est fusionné tant qu'une MR de la pile n'est pas prête
- **Critère** :
  - **Étant donné** la même pile, dont soit la MR du milieu a un pipeline rouge, soit `main` a reçu un commit que la pile ne contient pas
  - **Quand** `harness merge-stack 3` est lancé
  - **Alors**
    - la sortie nomme l'obstacle (`CHECKS FAIL #2`, ou `OUTDATED #1`), se termine par `FAIL nothing merged`, et le code de sortie est 1
    - aucune MR n'est fusionnée
- **Test** : `hooks/tests/merge-stack.bats` › `FUS-003 nothing is merged when a merge request of the stack has a red pipeline` ; `hooks/tests/merge-stack.bats` › `FUS-003 nothing is merged when the default branch has moved under the stack`

### FUS-004 — La fusion s'arrête à la MR qui résiste et dit ce qui reste
- **Critère** :
  - **Étant donné** la même pile, dont la forge refuse de fusionner la MR du milieu
  - **Quand** `harness merge-stack 3` est lancé
  - **Alors**
    - la sortie contient `STOPPED #2`, `MERGED: #1`, `REMAINING: #2 #3`, se termine par `FAIL stack partly merged`, et le code de sortie est 1
    - la MR du bas est fusionnée, les deux autres restent ouvertes
- **Test** : `hooks/tests/merge-stack.bats` › `FUS-004 the merge stops at the request that resists and says what is left`

### FUS-005 — Une branche par défaut qui n'est pas le code du haut de la pile est signalée
- **Critère** :
  - **Étant donné** la même pile, et un tiers qui pousse un commit sur `main` pendant la fusion
  - **Quand** `harness merge-stack 3` est lancé
  - **Alors** la sortie contient `TREE DIFFERS` avec le nom du fichier en écart, se termine par `FAIL stack merged, main differs from the top of the stack`, et le code de sortie est 1
- **Test** : `hooks/tests/merge-stack.bats` › `FUS-005 a default branch that is not the code of the top is reported`

### FUS-006 — Une branche par défaut rouge après la fusion est signalée
- **Critère** :
  - **Étant donné** la même pile, et un pipeline de `main` qui échoue sur le commit fusionné
  - **Quand** `harness merge-stack 3` est lancé
  - **Alors** la sortie contient `TREE IDENTICAL` puis `CHECKS FAIL main`, se termine par `FAIL stack merged, main is red`, et le code de sortie est 1
- **Test** : `hooks/tests/merge-stack.bats` › `FUS-006 a red default branch after the merge is reported`

### FUS-007 — Sans pipeline, la commande de validation fait foi
- **Critère** :
  - **Étant donné** la même pile, sur un dépôt dont `main` n'a aucun pipeline
  - **Quand** `harness merge-stack 3` est lancé avec `--validate '<commande>'`, puis sans
  - **Alors**
    - avec une commande qui réussit sur le code fusionné : `VALIDATION PASS`, dernière ligne `PASS`, code de sortie 0, et `git worktree list` est celui d'avant
    - avec une commande qui échoue : `VALIDATION FAIL`, dernière ligne `FAIL stack merged, main is red`, code de sortie 1
    - sans commande : `CHECKS NONE`, dernière ligne `FAIL stack merged, main not verified`, code de sortie 1
- **Test** : `hooks/tests/merge-stack.bats` › `FUS-007 without a pipeline the validation command decides` ; `hooks/tests/merge-stack.bats` › `FUS-007 a failing validation command is reported` ; `hooks/tests/merge-stack.bats` › `FUS-007 without a pipeline nor a validation command the merge is not verified`

### FUS-008 — Un essai à blanc vérifie la pile sans rien fusionner
- **Critère** :
  - **Étant donné** la même pile
  - **Quand** `harness merge-stack --dry-run 3` est lancé
  - **Alors**
    - la sortie contient `READY`, se termine par `PASS nothing merged (dry run)`, et le code de sortie est 0
    - aucune MR n'est fusionnée et `main` n'a pas bougé
- **Test** : `hooks/tests/merge-stack.bats` › `FUS-008 a dry run checks the stack and merges nothing`

### FUS-009 — Fusionner une pile passe par l'accord de l'utilisateur
- **Critère** :
  - **Étant donné** le hook `never-merge`
  - **Quand** il juge un appel Bash
  - **Alors**
    - `harness merge-stack 3` et `~/.local/bin/harness merge-stack --validate "mise run check" c` reçoivent `ask`
    - `harness merge-stack --dry-run 3` passe
    - `gh pr merge 3 --merge` et `glab mr merge 3` reçoivent `deny`
- **Test** : `hooks/tests/never-merge-stack.bats` › `FUS-009 merging a stack goes through the user's go`
- **Test** : `../harness/acceptance/parallel_test.go` › `TestPRV019_MutantsAreJudgedInParallel`

## sans-code-minimal — Un bloc qu'aucun test ne demande n'est plus jugé
Branche : `feat/sans-code-minimal` · Commande : `mise run test`

### PRV-020 — Un bloc qu'aucun test ne demande n'est plus jugé
- **Critère** :
  - **Étant donné** un commit d'acceptation, un commit `feat` qui ajoute dans un fichier de production un bloc dont le retrait laisse tous les tests verts, et un dossier de preuves dont le mutant tue le test du critère
  - **Quand** `harness gate green --proofs DIR --no-neutral` est lancé
  - **Alors**
    - la sortie ne contient aucune ligne `UNNEEDED` ni `MINIMAL`, et se termine par `PASS`
    - relancé avec `--no-minimal`, `--since`, `--unneeded-ok` ou `--spec`, la porte sort avec le code 2 et nomme l'option refusée
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV020_UnneededBlockIsNoLongerJudged` ; `../harness/acceptance/preuves_green_test.go` › `TestPRV020_RetiredOptionsAreRefused`

## couverture-go — Un bloc Go jamais exécuté fait échouer la porte
Branche : `feat/couverture-go` · Commande : `mise run test`

### COV-001 — Un bloc Go jamais exécuté fait échouer la porte
- **Critère** :
  - **Étant donné** un dépôt avec un `go.mod`, un fichier Go de production, et une tâche de test qui écrit un profil Go où un bloc de ce fichier a un compteur à 0 et les autres un compteur positif
  - **Quand** `harness gate coverage` est lancé avec ce profil
  - **Alors**
    - la sortie contient `UNCOVERED` avec le chemin du fichier dans le dépôt et les lignes du bloc, et se termine par `FAIL`
    - quand tous les blocs du profil ont un compteur positif, la sortie ne contient aucune ligne `UNCOVERED` et se termine par `PASS`
    - dans ce second cas, la sortie contient `ERROR RETURNS IGNORED 0`
    - quand le seul bloc au compteur 0 est un bloc sans instruction, la sortie ne contient aucune ligne `UNCOVERED` et se termine par `PASS`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV001_UncoveredGoBlockFails` ; `../harness/acceptance/couverture_test.go` › `TestCOV001_AllCoveredPasses` ; `../harness/acceptance/couverture_test.go` › `TestCOV001_EmptyBlockIsNotJudged`

### COV-003 — Les rapports de plusieurs suites se complètent
- **Critère** :
  - **Étant donné** deux rapports et trois blocs : le premier exécuté dans le premier rapport seulement, le deuxième dans le second seulement, le troisième à 0 dans les deux
  - **Quand** `harness gate coverage` est lancé avec `--coverage` donné deux fois
  - **Alors**
    - la sortie ne contient aucune ligne `UNCOVERED` pour le premier bloc
    - elle n'en contient aucune pour le deuxième
    - elle contient `UNCOVERED` pour le troisième et se termine par `FAIL`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV003_ReportsComplementEachOther`

### COV-007 — Un rapport non écrit ou une suite en échec ne passent pas
- **Critère** :
  - **Étant donné** un rapport resté d'une exécution précédente, où tout est couvert
  - **Quand** `harness gate coverage` est lancé avec une tâche qui n'écrit pas ce rapport, puis avec une tâche qui l'écrit et échoue
  - **Alors**
    - dans le premier cas, la sortie contient `COVERAGE missing` avec le chemin du rapport et se termine par `FAIL`
    - dans le second, elle contient `TASK FAILED` et se termine par `FAIL`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV007_ReportNotWrittenFails` ; `../harness/acceptance/couverture_test.go` › `TestCOV007_FailedTaskFails`

## couverture-python — Une ligne ou une branche Python jamais exécutée fait échouer la porte
Branche : `feat/couverture-python` · Commande : `mise run test`

### COV-002 — Une ligne ou une branche Python jamais exécutée fait échouer la porte
- **Critère** :
  - **Étant donné** un fichier Python de production et une tâche de test qui écrit un rapport LCOV où des lignes qui se suivent ont un compteur à 0, la première portant des branches au compteur `-`, et où une ligne exécutée porte une branche jamais prise
  - **Quand** `harness gate coverage` est lancé avec ce rapport
  - **Alors**
    - la sortie contient une seule ligne `UNCOVERED` pour ces lignes, avec le fichier et leur intervalle
    - elle contient `UNCOVERED BRANCH` avec le fichier et la ligne exécutée qui porte la branche jamais prise
    - elle ne contient aucune ligne `UNCOVERED BRANCH` pour la ligne jamais exécutée
    - elle se termine par `FAIL`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV002_UncoveredPythonLinesAndBranch`

### COV-008 — `# pragma: no cover` est refusé
- **Critère** :
  - **Étant donné** un fichier Python de production dont une ligne porte `# pragma: no cover` et une autre `# pragma: no branch`, et un rapport où tout ce qui est mesuré est couvert
  - **Quand** `harness gate coverage` est lancé
  - **Alors**
    - la sortie contient `PRAGMA` avec le fichier et la ligne du `no cover`, et se termine par `FAIL`
    - elle contient `PRAGMA` avec le fichier et la ligne du `no branch`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV008_PragmaIsRefused`

## couverture-exceptions — Une exception s'écrit, avec sa raison, hors du code
Branche : `feat/couverture-exceptions` · Commande : `mise run test`

### COV-005 — Une exception écrite est acceptée, une exception périmée fait échouer
- **Critère** :
  - **Étant donné** un bloc jamais exécuté, et `COVERAGE.md` qui porte, pour son fichier, une entrée avec le texte de sa première ligne et une raison
  - **Quand** `harness gate coverage` est lancé
  - **Alors**
    - la sortie contient `ACCEPTED` avec le fichier, les lignes du bloc et la raison, aucune ligne `UNCOVERED`, et se termine par `PASS`
    - quand le bloc est exécuté et que l'entrée reste, la sortie contient `STALE EXCEPTION` avec le fichier et le texte, et se termine par `FAIL`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV005_WrittenExceptionIsAccepted`, `TestCOV005_StaleExceptionFails`

### COV-006 — Un fichier Go ou Python que nul rapport ne mesure est signalé
- **Critère** :
  - **Étant donné** un dépôt qui suit un fichier Python de production absent de tout rapport, un fichier de test Python absent lui aussi, un fichier Go de production absent lui aussi et qui ne déclare aucune fonction, et un fichier YAML
  - **Quand** `harness gate coverage` est lancé
  - **Alors**
    - la sortie contient `NOT MEASURED` avec le fichier de production et se termine par `FAIL`
    - elle ne contient aucune ligne `NOT MEASURED` pour le fichier de test ni pour le fichier YAML
    - elle n'en contient aucune pour le fichier Go sans fonction
    - elle contient une ligne `NOT JUDGED` qui compte le fichier YAML
    - quand `COVERAGE.md` porte une entrée `fichier entier` pour le fichier de production, la sortie contient `ACCEPTED` pour lui et se termine par `PASS`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV006_UnmeasuredFileFails`, `TestCOV006_WholeFileExceptionIsAccepted`

### COV-009 — `--baseline` écrit la dette de départ
- **Critère** :
  - **Étant donné** un dépôt avec deux blocs jamais exécutés et aucun `COVERAGE.md`
  - **Quand** `harness gate coverage --baseline` est lancé, puis `harness gate coverage` sans l'option
  - **Alors**
    - après le premier lancement, `COVERAGE.md` porte, sous le fichier de chaque bloc, une entrée avec le texte de sa première ligne et la raison `dette initiale` ; ce lancement se termine par `PASS`
    - le second lancement imprime `ACCEPTED` pour chacun, aucune ligne `UNCOVERED`, et se termine par `PASS`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV009_BaselineWritesTheDebt`, `TestCOV009_BaselineWritesNothingWhenTaskFails`

## couverture-remontees — Une remontée d'erreur Go est écartée et comptée
Branche : `feat/couverture-remontees` · Commande : `mise run test`

### COV-004 — Une remontée d'erreur Go est écartée et comptée
- **Critère** :
  - **Étant donné** un fichier Go avec deux blocs jamais exécutés : le corps d'un `if err != nil` qui ne contient qu'un `return`, et le corps d'un `if rc != 0` qui ne contient qu'un `return`
  - **Quand** `harness gate coverage` est lancé
  - **Alors**
    - la sortie ne contient aucune ligne `UNCOVERED` pour le premier bloc, et contient `ERROR RETURNS IGNORED 1`
    - elle contient `UNCOVERED` pour le second et se termine par `FAIL`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV004_ErrorReturnIsIgnoredAndCounted`

## couverture-revue-finale — La revue finale resserre deux règles
Branche : `fix/couverture-revue-finale` · Commande : `mise run test`

### COV-010 — Un `return` qui avale l'erreur est jugé
- **Critère** :
  - **Étant donné** un fichier Go avec deux blocs jamais exécutés, chacun le corps entier d'un `if err != nil` fait d'un seul `return` : le premier rend l'erreur testée, le second ne la nomme dans aucun de ses résultats
  - **Quand** `harness gate coverage` est lancé
  - **Alors**
    - la sortie ne contient aucune ligne `UNCOVERED` pour le premier bloc, et contient `ERROR RETURNS IGNORED 1`
    - elle contient `UNCOVERED` pour le second et se termine par `FAIL`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV010_ReturnSwallowingTheErrorIsJudged`

### COV-011 — Les écritures de pragma que coverage.py reconnaît sont refusées
- **Critère** :
  - **Étant donné** un fichier Python de production dont trois lignes portent `# pragma no cover`, `# PRAGMA: NO COVER` et `#  pragma:no branch`, et un rapport où tout ce qui est mesuré est couvert
  - **Quand** `harness gate coverage` est lancé
  - **Alors**
    - la sortie contient `PRAGMA` avec le fichier et la ligne du `# pragma no cover`, et se termine par `FAIL`
    - elle contient `PRAGMA` avec le fichier et la ligne du `# PRAGMA: NO COVER`
    - elle contient `PRAGMA` avec le fichier et la ligne du `#  pragma:no branch`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV011_RecognizedPragmaSpellingsAreRefused`

## couverture-fichier-vide — Un fichier Go dont les fonctions sont vides est mesuré
Branche : `fix/couverture-fichier-vide` · Commande : `mise run test`

### COV-012 — Un fichier Go dont les fonctions sont vides est mesuré
- **Critère** :
  - **Étant donné** un fichier Go de production dont la seule fonction a un corps vide, et un profil Go qui ne porte pour ce fichier qu'un bloc sans instruction
  - **Quand** `harness gate coverage` est lancé avec ce profil
  - **Alors**
    - la sortie ne contient aucune ligne `NOT MEASURED` pour ce fichier
    - elle ne contient aucune ligne `UNCOVERED` et se termine par `PASS`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV012_FileOfEmptyFunctionsIsMeasured`

## couverture-exclusions — Une exclusion dans la configuration de coverage.py est refusée
Branche : `feat/couverture-exclusions` · Commande : `mise run test`

### COV-013 — Une clé d'exclusion dans `.coveragerc` est refusée
- **Critère** :
  - **Étant donné** un fichier Python de production, un rapport où tout ce qui est mesuré est couvert, et un `.coveragerc` suivi par git dont la section `[report]` porte une clé `exclude_lines` et une clé `partial_branches_always`
  - **Quand** `harness gate coverage` est lancé
  - **Alors**
    - la sortie contient `PRAGMA` avec `.coveragerc` et la ligne de `exclude_lines`, et se termine par `FAIL`
    - elle contient `PRAGMA` avec `.coveragerc` et la ligne de `partial_branches_always`
    - quand `.coveragerc` ne porte que la section `[run]` (`branch`, `parallel`, `relative_files`, `sigterm`, `source`), la sortie ne contient aucune ligne `PRAGMA` et se termine par `PASS`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV013_ExclusionKeysInCoveragercAreRefused`, `TestCOV013_RunOnlyCoveragercIsAccepted`

### COV-014 — Les autres fichiers de configuration de coverage.py sont lus
- **Critère** :
  - **Étant donné** un fichier Python de production, un rapport où tout ce qui est mesuré est couvert, et trois fichiers suivis par git : `pyproject.toml` dont la table `[tool.coverage.report]` porte `exclude_also`, `setup.cfg` dont la section `[coverage:report]` porte `partial_branches`, `tox.ini` dont la section `[coverage:report]` porte `partial_also`
  - **Quand** `harness gate coverage` est lancé
  - **Alors**
    - la sortie contient `PRAGMA` avec `pyproject.toml` et la ligne de `exclude_also`, et se termine par `FAIL`
    - elle contient `PRAGMA` avec `setup.cfg` et la ligne de `partial_branches`
    - elle contient `PRAGMA` avec `tox.ini` et la ligne de `partial_also`
    - quand `pyproject.toml` ne porte aucune table `tool.coverage` (`[project]`, `[tool.hatch.build.targets.wheel]`, `[dependency-groups]`) et que les deux autres fichiers sont absents, la sortie ne contient aucune ligne `PRAGMA` et se termine par `PASS`
- **Test** : `../harness/acceptance/couverture_test.go` › `TestCOV014_OtherConfigurationFilesAreRead`, `TestCOV014_PyprojectWithoutCoverageTableIsAccepted`

## forge-hote — Le dépôt dit sa forge, un hôte inconnu est refusé
Branche : `feat/forge-hote` · Commande : `mise run test`

### FGL-010 — Un hôte de forge inconnu est refusé
- **Critère** :
  - **Étant donné** un dépôt dont `origin` est sur `inconnu.example.test`, absent de `forges`
  - **Quand** `harness merge-stack 1`
  - **Alors**
    - la sortie se termine par un `FAIL` qui nomme `inconnu.example.test` et le fichier `forges`
    - le code de sortie est 1
    - ni le faux `gh` ni le faux `glab` n'ont été appelés
- **Test** : `hooks/tests/merge-stack-forge.bats` › `FGL-010 an unknown forge host is refused before any forge tool is called`

## fusion-gitlab — `harness merge-stack` sur GitLab
Branche : `feat/fusion-gitlab` · Commande : `mise run test`

### FGL-001 — Une pile GitLab se fusionne du bas vers le haut et la cible porte le code du haut
- **Critère** :
  - **Étant donné** un dépôt dont `origin` est sur `forge.example.test`, déclaré `glab` dans `forges` ; une pile !1 ← !2 ← !3 sur `main`, ouvertes, vertes, `mergeable`
  - **Quand** `harness merge-stack 3`
  - **Alors**
    - la sortie contient `MERGED #1`, `MERGED #2`, `MERGED #3` dans cet ordre
    - les trois MR sont à l'état `merged`
    - l'arbre de `main` sur le dépôt est celui de la tête de !3
    - la sortie se termine par `PASS` et le code de sortie est 0
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-001 a GitLab stack is merged bottom first and main holds the code of its top`

### FGL-002 — Une MR GitLab seule se fusionne sans celles du dessus
- **Critère** :
  - **Étant donné** la pile de FGL-001
  - **Quand** `harness merge-stack 1`
  - **Alors**
    - !1 est `merged`
    - !2 et !3 restent `opened`
    - la sortie se termine par `PASS`
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-002 a single GitLab merge request is merged without the ones above it`

### FGL-003 — Rien n'est fusionné tant qu'une MR GitLab n'est pas prête
- **Critère** :
  - **Étant donné** la pile de FGL-001, avec sur !2, dans un cas chacun : pipeline `failed` ; `head_pipeline` sur un autre sha que le sha de tête ; `has_conflicts` vrai ; `detailed_merge_status` à `not_approved` ; `main` qui a avancé sous la pile
  - **Quand** `harness merge-stack 3`
  - **Alors**
    - aucune MR n'est fusionnée
    - la sortie nomme !2 et la cause (`CHECKS FAIL #2`, `CONFLICTS #2`, `NOT MERGEABLE #2: not_approved`, `OUTDATED #1`)
    - la sortie se termine par `FAIL nothing merged` et le code de sortie est 1
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-003 nothing is merged when a GitLab merge request has a failed pipeline`, `FGL-003 nothing is merged when the pipeline of a GitLab merge request is on another sha`, `FGL-003 nothing is merged when a GitLab merge request has conflicts`, `FGL-003 nothing is merged when GitLab does not say mergeable`, `FGL-003 nothing is merged when main has moved under the GitLab stack`

### FGL-004 — La fusion GitLab est figée sur le sha vérifié et supprime la branche source
- **Critère** :
  - **Étant donné** la pile de FGL-001
  - **Quand** `harness merge-stack 3`
  - **Alors**
    - chaque appel de fusion reçu par le faux `glab` porte `sha` égal au sha de tête de la MR lu avant la fusion
    - chaque appel de fusion porte `should_remove_source_branch=true`
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-004 a GitLab merge is frozen on the verified sha and removes the source branch`

### FGL-005 — La fusion GitLab s'arrête à la MR qui résiste et dit ce qui reste
- **Critère** :
  - **Étant donné** la pile de FGL-001, et le faux `glab` refuse la fusion de !2 (tête qui a bougé)
  - **Quand** `harness merge-stack 3`
  - **Alors**
    - la sortie contient `STOPPED #2`, `MERGED: #1`, `REMAINING: #2 #3`
    - la sortie se termine par `FAIL stack partly merged` et le code de sortie est 1
    - !1 est `merged`, !2 et !3 restent `opened`
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-005 the GitLab merge stops at the request that resists and says what is left`

### FGL-006 — Après fusion GitLab, une cible qui n'est pas le code du haut est signalée
- **Critère** :
  - **Étant donné** la pile de FGL-001, et un commit poussé sur `main` par un tiers pendant la fusion
  - **Quand** `harness merge-stack 3`
  - **Alors**
    - la sortie contient `TREE DIFFERS`
    - la sortie se termine par `FAIL stack merged, main differs from the top of the stack`
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-006 after a GitLab merge a main that is not the code of the top is reported`

### FGL-007 — Après fusion GitLab, une cible rouge est signalée
- **Critère** :
  - **Étant donné** la pile de FGL-001, et le pipeline du commit de fusion sur `main` `failed`
  - **Quand** `harness merge-stack 3`
  - **Alors**
    - la sortie contient `CHECKS FAIL main`
    - la sortie se termine par `FAIL stack merged, main is red` et le code de sortie est 1
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-007 after a GitLab merge a red main is reported`

### FGL-008 — Sans pipeline GitLab, la commande de validation fait foi
- **Critère** :
  - **Étant donné** la pile de FGL-001, et aucun pipeline sur `main` après fusion
  - **Quand** `harness merge-stack --validate <commande> 3`, avec une commande qui réussit, puis une qui échoue, puis sans `--validate`
  - **Alors**
    - commande qui réussit : `VALIDATION PASS`, puis `PASS`
    - commande qui échoue : `VALIDATION FAIL`, puis `FAIL stack merged, main is red`
    - sans `--validate` : `CHECKS NONE`, puis `FAIL stack merged, main not verified`
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-008 without a GitLab pipeline a validation command that succeeds decides`, `FGL-008 without a GitLab pipeline a validation command that fails decides`, `FGL-008 without a GitLab pipeline nor a validation command the merge is not verified`

### FGL-009 — Un essai à blanc GitLab vérifie la pile sans rien fusionner
- **Critère** :
  - **Étant donné** la pile de FGL-001
  - **Quand** `harness merge-stack --dry-run 3`
  - **Alors**
    - la sortie contient `READY`
    - aucune MR n'est fusionnée et le faux `glab` n'a reçu aucun appel de fusion ni de repointage
    - la sortie se termine par `PASS nothing merged (dry run)`
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-009 a GitLab dry run checks the stack and merges nothing`

### FGL-013 — Une MR GitLab se désigne par `!n` ou par son URL, pas par l'URL d'un autre projet
- **Critère** :
  - **Étant donné** la pile de FGL-001
  - **Quand** `harness merge-stack --dry-run '!3'`, puis `harness merge-stack --dry-run <web_url de !3>`, puis `harness merge-stack --dry-run <URL de MR d'un autre projet sur forge.example.test>`
  - **Alors**
    - avec `!3`, la sortie se termine par `PASS nothing merged (dry run)`
    - avec l'URL de !3, la sortie se termine par `PASS nothing merged (dry run)`
    - avec l'URL d'un autre projet, la sortie se termine par un `FAIL` et le code de sortie est 1
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-013 a GitLab merge request is designated by !n or by its URL, not by the URL of another project`

### FGL-014 — Sur un projet à pipelines de MR, seul un pipeline de MR vert compte
- **Critère** :
  - **Étant donné** la pile de FGL-001, le projet lance des pipelines de MR (une source `merge_request_event` dans les pipelines de !2), et le `head_pipeline` de !2 est un pipeline `push` en `success` sur le sha de tête
  - **Quand** `harness merge-stack 3`
  - **Alors**
    - aucune MR n'est fusionnée
    - la sortie nomme !2 (`CHECKS FAIL #2` ou `CHECKS UNKNOWN #2`)
    - la sortie se termine par `FAIL nothing merged`
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-014 on a project with merge request pipelines only a green merge request pipeline counts`

### FGL-015 — Après un repointage, la fusion GitLab attend que GitLab se prononce
- **Critère** :
  - **Étant donné** la pile de FGL-001, et après la fusion de !1 le faux `glab` répond `detailed_merge_status: checking` pour !2 aux deux premières lectures, puis `mergeable`
  - **Quand** `harness merge-stack 3`
  - **Alors**
    - les trois MR sont `merged`
    - aucun appel de fusion de !2 n'est reçu tant que le statut est `checking`
    - la sortie se termine par `PASS`
- **Test** : `hooks/tests/merge-stack-gitlab.bats` › `FGL-015 after a retarget the GitLab merge waits for GitLab to settle`
