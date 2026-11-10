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
- **Test** : `agents/bin/tests/test_acceptance_mutant_mise.py` › `test_PRV008_mutant_of_code_task_is_judged` ; `agents/bin/tests/test_acceptance_mutant_mise.py` › `test_PRV008_mutant_of_test_task_is_refused`
