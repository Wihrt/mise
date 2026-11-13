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

## code-minimal — Code minimal : un bloc qu'aucun test ne demande est refusé
Branche : `feat/code-minimal` · Commande : `mise run test`

### PRV-009 — Un bloc de code qu'aucun test ne demande est refusé
- **Critère** :
  - **Étant donné** un commit d'acceptation et un commit `feat` qui ajoute, dans un fichier de production, deux blocs séparés : l'un sans lequel le test du critère échoue, l'autre dont le retrait laisse tous les tests verts
  - **Quand** `harness gate green --proofs` est lancé
  - **Alors**
    - la sortie contient `UNNEEDED` avec le fichier et les lignes du second bloc, aucune ligne `UNNEEDED` pour le premier, et se termine par `FAIL`
    - relancé avec `--unneeded-ok` pour ce bloc, elle contient `UNNEEDED ACCEPTED` avec le même fichier et les mêmes lignes, aucune autre ligne `UNNEEDED`, et se termine par `PASS`
    - quand le commit `feat` supprime aussi une ligne existante d'un fichier de production et ajoute un fichier de test, aucune ligne `UNNEEDED` ne nomme la suppression ni le fichier de test
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV009_UnneededBlockIsRefused` ; `../harness/acceptance/preuves_green_test.go` › `TestPRV009_UnneededOKAcceptsTheBlock` ; `../harness/acceptance/preuves_green_test.go` › `TestPRV009_DeletionAndTestFileAreNotJudged`

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

### PRV-013 — Le code minimal se juge depuis le début de la tâche
- **Critère** :
  - **Étant donné** un slug qui a deux commits d'acceptation, et un bloc de code de production qu'aucun test ne demande, ajouté entre les deux
  - **Quand** `harness gate green --proofs DIR` est lancé avec `--test <second commit d'acceptation>` et `--since <premier commit d'acceptation>`
  - **Alors** la sortie contient `UNNEEDED` avec le fichier et les lignes de ce bloc, et se termine par `FAIL`
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV013_UnneededBlockAddedBetweenTwoAcceptanceCommits`

### RPL-003 — Une tâche qui échoue sans mutant arrête le rejeu
- **Critère** :
  - **Étant donné** deux commits dont le premier fait échouer la tâche sans aucun mutant
  - **Quand** `harness gate replay --base <premier> --head <second>` est lancé
  - **Alors**
    - la sortie contient `BASELINE NOT GREEN base`, aucune ligne `KEPT`, et se termine par `FAIL`
    - relancé en échangeant `--base` et `--head`, elle contient `BASELINE NOT GREEN head`, aucune ligne `KEPT`, et se termine par `FAIL`
- **Test** : `../harness/acceptance/rejeu_test.go` › `TestRPL003_FailingTaskWithoutMutantStopsTheReplay` ; `../harness/acceptance/rejeu_test.go` › `TestRPL003_SwappedBaseAndHeadNamesHead`

## spec-hors-code — La spec n'est pas jugée comme du code
Branche : `feat/spec-hors-code` · Commande : `mise run test`

### PRV-014 — La spec n'est pas jugée comme du code
- **Critère** :
  - **Étant donné** un slug qui a deux commits d'acceptation et, entre les deux, un commit qui ajoute des lignes au fichier de spec du dépôt
  - **Quand** `harness gate green --proofs --since <premier commit d'acceptation> --spec <fichier de spec>` est lancé
  - **Alors** la sortie ne contient aucune ligne `UNNEEDED` qui nomme le fichier de spec, et se termine par `PASS`
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV014_SpecLinesAddedBetweenTwoAcceptanceCommitsAreNotUnneeded`

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
## preuves-optionnelles — Le changement neutre et le code minimal se sautent sur demande
Branche : `feat/harness-preuves-optionnelles` · Commande : `mise run test`

### PRV-017 — Le changement neutre se saute sur demande
- **Critère** :
  - **Étant donné** un commit d'acceptation, un commit `feat` vert, et un dossier de preuves qui contient un mutant tuant le test du critère et aucun changement neutre
  - **Quand** `harness gate green --proofs` est lancé avec `--no-neutral`
  - **Alors**
    - la sortie contient `KILLED` pour le critère et `NEUTRAL SKIPPED`, aucune ligne `NEUTRAL MISSING`, et se termine par `PASS`
    - relancé sans `--no-neutral`, elle contient `NEUTRAL MISSING` pour le critère et se termine par `FAIL`
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV017_NeutralChangeIsSkippedOnRequest`

### PRV-018 — Le code minimal se saute sur demande
- **Critère** :
  - **Étant donné** un commit d'acceptation et un commit `feat` qui ajoute, dans un fichier de production, un bloc dont le retrait laisse tous les tests verts
  - **Quand** `harness gate green --proofs` est lancé avec `--no-minimal`
  - **Alors**
    - la sortie contient `MINIMAL SKIPPED`, aucune ligne `UNNEEDED`, et se termine par `PASS`
    - relancé sans `--no-minimal`, elle contient `UNNEEDED` avec le fichier et les lignes du bloc et se termine par `FAIL`
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV018_MinimalCodeIsSkippedOnRequest`

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
    - relancé avec `--no-minimal`, la porte sort avec le code 2 (option inconnue)
- **Test** : `../harness/acceptance/preuves_green_test.go` › `TestPRV020_UnneededBlockIsNoLongerJudged` ; `../harness/acceptance/preuves_green_test.go` › `TestPRV020_RetiredOptionsAreRefused`
