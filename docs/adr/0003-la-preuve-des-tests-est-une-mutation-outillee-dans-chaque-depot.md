# 3. La preuve des tests est une mutation outillée dans chaque dépôt

Date: 2026-11-27

## Status

Proposed

## Context

- Aujourd'hui, c'est le harnais qui prouve la valeur d'un test. `harness gate red` montre le test rouge. L'agent `mutant-writer` écrit ensuite un mutant par clause de critère, et `harness gate green --proofs` vérifie qu'il est tué (`claude/skills/stacked-tdd-mr/SKILL.md:11-14`). Chaque maillon dépend d'un agent, et la chaîne est lente.
- Des outils de mutation standard existent pour les trois piles (vérifié le 2026-11-27) :
  - gremlins v0.6.0 pour Go, avec un mode diff (`--diff`). Ses seuils natifs sortent en 0 même sous le seuil (essai).
  - pytest-gremlins 1.15.0 pour Python. C'est un plugin pytest rapide, mais sans seuil ni mode diff. Le score se lit dans `coverage/gremlins/gremlins.json`. mutmut (ni seuil ni diff) et cosmic-ray (lent) ont été écartés.
  - helm-mutation-test v1.2.0 pour Helm, avec un mode diff (`--since`) et la détection des mutants équivalents.
- gremlins et pytest-gremlins ne mutent que le code exercé dans leur propre processus. Un test qui lance le binaire ou un conteneur ne compte pas. Essai gremlins : `Killed: 0, Lived: 0, Not covered: 2`, avec un code de sortie 0.
- La porte de couverture `harness gate coverage` est déjà déterministe et mesure les tests de frontière (ADR 2).

Options :
1. Les outils dans chaque dépôt (tâche `mutation`, verdict en CI), et la preuve TDD du harnais disparaît. Compromis : plus rien ne vérifie que le test précède le code.
2. Le harnais enveloppe les outils. Compromis : du code maison reste, et un dépôt dépend du harnais pour sa preuve.
3. L'option 1, plus une porte red minimale. Compromis : la discipline « test d'abord » reste, au prix d'un reste de harnais.

## Decision

Chaque dépôt Go, Python ou Helm déclarera dans son `mise.toml` une tâche `mutation`, lancée par `check` et donc par la CI. Elle échoue dès qu'un mutant du code changé depuis `MUTATION_BASE` survit ou n'est atteint par aucun test. Le verdict est lu par `jq` dans le rapport de l'outil, jamais dans son code de sortie. Les outils sont :
- Go : gremlins, avec `--diff` ;
- Python : pytest-gremlins sur les fichiers changés ;
- Helm : helm-mutation-test, avec `--since`.

En Go et en Python, la mutation mesure les tests unitaires, qui deviennent obligatoires. Les tests d'acceptation restent à la frontière, et la porte de couverture prouve qu'ils atteignent tout le code.

Nous retirerons du harnais :
- les commandes `gate red`, `green`, `characterize` et `replay` ;
- le domaine `tdd` ;
- l'agent `mutant-writer` ;
- les fichiers `TESTS.md`.

Nous garderons `gate coverage`. Un mutant équivalent s'exclut par la configuration de l'outil, et l'exclusion est consignée.

## Consequences

- **Plus facile.** Un dépôt prouve ses tests sans Claude ni harnais (`mise run mutation`). Le harnais perd une large part de son code, et une tâche demande moins d'allers-retours entre agents.
- **Plus difficile.**
  - L'ordre « test avant code » n'est plus vérifié.
  - Il faut écrire deux suites : unitaire et de frontière.
  - En Python, un fichier changé est muté en entier, anciens survivants compris, et pytest-gremlins a moins d'un an.
  - gremlins ne détecte pas les mutants équivalents, d'où des exclusions à consigner.
  - helm-mutation-test est jeune et n'a qu'un mainteneur.
- **À revoir si :**
  - la tâche `mutation` d'un dépôt dépasse le temps de CI acceptable ;
  - les exclusions d'équivalents se multiplient au point de vider la règle de son sens ;
  - un des trois outils n'est plus maintenu, ou corrige ses seuils natifs.
