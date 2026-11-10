# Tests

## forme — Forme des preuves du harnais
Branche : `feat/forme` · Commande : `mise run test`

### FRM-001 — Le critère est écrit en toutes lettres dans l'entrée
- **Critère** :
  - **Étant donné** un `TESTS.md` dont l'entrée `X-001` du slug jugé ne porte qu'un renvoi (`**Critère** : docs/specs/x.md › X-001`), et dont une section d'un autre slug est à l'ancien format
  - **Quand** `tdd-gate red`, `green` ou `characterize` est lancé sur le slug
  - **Alors** la sortie contient `INCOMPLETE X-001` et se termine par `FAIL`, sans aucune ligne `INCOMPLETE` pour l'autre section
- **Test** : `agents/bin/tests/test_acceptance_forme.py` › `test_FRM001_red_incomplete_entry` ; `agents/bin/tests/test_acceptance_forme.py` › `test_FRM001_green_incomplete_entry` ; `agents/bin/tests/test_acceptance_forme.py` › `test_FRM001_characterize_incomplete_entry`

### FRM-003 — Un rapport absent dit quoi faire
- **Critère** :
  - **Étant donné** une tâche de test qui sort à 0 sans écrire de rapport
  - **Quand** `tdd-gate characterize` est lancé
  - **Alors** la sortie contient une ligne `REPORT missing` qui nomme la tâche et le chemin `.tdd/junit.xml`, et se termine par `FAIL`
- **Test** : `agents/bin/tests/test_acceptance_forme.py` › `test_FRM003_characterize_report_missing`

### FRM-005 — La spec jugée est celle qui est commitée
- **Critère** :
  - **Étant donné** une spec située dans le dépôt, avec une modification non commitée
  - **Quand** `tdd-gate red` ou `characterize` est lancé avec `--spec`
  - **Alors** la sortie contient `SPEC NOT COMMITTED` et se termine par `FAIL`
- **Test** : `agents/bin/tests/test_acceptance_forme.py` › `test_FRM005_red_spec_not_committed` ; `agents/bin/tests/test_acceptance_forme.py` › `test_FRM005_characterize_spec_not_committed`

## branchement — Branchement de la tâche de test
Branche : `feat/branchement` · Commande : `mise run test`

### FRM-004 — La tâche de test tourne quelque part
- **Critère** :
  - **Étant donné** un slug dont la `Commande` est `mise run test:render`, tâche citée par aucun hook de commit ni job CI du dépôt, ni dépendance d'une tâche qui l'est
  - **Quand** `tdd-gate characterize` ou `green` est lancé
  - **Alors**
    - la sortie contient `NOT WIRED test:render` et se termine par `FAIL`
    - une fois la tâche citée dans `.pre-commit-config.yaml` ou `.gitlab-ci.yml`, la ligne `NOT WIRED` n'apparaît plus
    - quand le hook ne cite que `mise run check` et que `test:render` est une dépendance (`depends`) de `check` dans `mise.toml`, la ligne `NOT WIRED` n'apparaît pas
- **Test** : `agents/bin/tests/test_acceptance_branchement.py` › `test_FRM004_green_task_not_wired` ; `agents/bin/tests/test_acceptance_branchement.py` › `test_FRM004_characterize_task_not_wired` ; `agents/bin/tests/test_acceptance_branchement.py` › `test_FRM004_green_task_wired` ; `agents/bin/tests/test_acceptance_branchement.py` › `test_FRM004_characterize_task_wired` ; `agents/bin/tests/test_acceptance_branchement.py` › `test_FRM004_green_task_wired_by_dependency`

## mutant-mise — Mutant du code écrit dans une tâche mise
Branche : `feat/mutant-mise` · Commande : `mise run test`

### PRV-008 — Un mutant du code écrit dans une tâche mise est jugé
- **Critère** :
  - **Étant donné** un dépôt dont le code testé est le corps de la tâche `[tasks."bootstrap:agent"]` de `mise.toml`, jugé par la tâche `test:bootstrap`
  - **Quand** `tdd-gate characterize` reçoit un mutant qui modifie le corps de `bootstrap:agent`, puis un mutant qui modifie le corps de `test:bootstrap`
  - **Alors**
    - le premier est appliqué et jugé (`KILLED` ou `SURVIVED`)
    - le second est refusé par `MUTANT INVALID … touches test task test:bootstrap`
    - un mutant d'un autre fichier que `mise.toml`, dont le hunk porte sur des numéros de ligne qui sont, dans `mise.toml`, ceux du corps de `test:bootstrap`, est appliqué et jugé (`KILLED` ou `SURVIVED`)
- **Test** : `agents/bin/tests/test_acceptance_mutant_mise.py` › `test_PRV008_mutant_of_code_task_is_judged` ; `agents/bin/tests/test_acceptance_mutant_mise.py` › `test_PRV008_mutant_of_test_task_is_refused` ; `agents/bin/tests/test_acceptance_mutant_mise.py` › `test_PRV008_mutant_of_other_file_is_judged`

## preuve-sensible — Un mutant par clause du « Alors »
Branche : `feat/preuve-sensible` · Commande : `mise run test`

### PRV-001 — Un mutant par clause est exigé
- **Critère** :
  - **Étant donné** un slug dont le critère `X-001` a un « Alors » à deux clauses, et un dossier de preuves qui contient `X-001.patch` et `X-001.neutral.patch` mais pas `X-001.2.patch`
  - **Quand** `tdd-gate characterize` est lancé sur le commit d'acceptation
  - **Alors** la sortie contient `MUTANT MISSING X-001 clause 2` et se termine par `FAIL`
- **Test** : `agents/bin/tests/test_acceptance_preuve_sensible.py` › `test_PRV001_missing_mutant_for_clause_2`

### PRV-002 — Une clause qu'aucun test ne défend est refusée
- **Critère** :
  - **Étant donné** le même critère à deux clauses, les deux mutants présents, et un test qui n'échoue que sous le premier
  - **Quand** `tdd-gate characterize` est lancé
  - **Alors** la sortie contient `KILLED X-001 clause 1` et `SURVIVED X-001 clause 2`, et se termine par `FAIL`
- **Test** : `agents/bin/tests/test_acceptance_preuve_sensible.py` › `test_PRV002_clause_without_defending_test_survives`
