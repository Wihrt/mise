# 3. La preuve des tests est une mutation outillée dans chaque dépôt

Date: 2026-11-27

## Status

Proposed

## Context

- La valeur d'un test est aujourd'hui prouvée par le harnais. `harness gate red` montre le test rouge. L'agent `mutant-writer` écrit ensuite un mutant par clause de critère, et `harness gate green --proofs` vérifie qu'il est tué (`claude/skills/stacked-tdd-mr/SKILL.md:11-14`). Chaque maillon dépend d'un agent, et la chaîne est lente.
- Des outils de mutation standard existent pour les trois piles principales (vérifié le 2026-11-27) :
  - gremlins v0.6.0 pour Go, avec un seuil natif (`--threshold-efficacy`) et un mode diff (`--diff`) ;
  - pytest-gremlins 1.15.0 pour Python : plugin pytest rapide (parallèle, cache), sans seuil ni mode diff natifs ; le score se lit dans `coverage/gremlins/gremlins.json` et la liste des fichiers changés se passe à `--gremlin-targets`. mutmut (ni seuil ni diff) et cosmic-ray (lent) ont été écartés ;
  - helm-mutation-test v1.2.0 pour Helm, avec un seuil natif (`--threshold`), un mode diff (`--since`) et la détection des mutants équivalents.
- La porte de couverture `harness gate coverage` est déjà déterministe (ADR 2).

Options :
1. Les outils dans chaque dépôt (tâche `mutation`, verdict en CI) ; la preuve TDD du harnais disparaît. Compromis : plus rien ne vérifie que le test précède le code.
2. Le harnais enveloppe les outils. Compromis : du code maison reste, et un dépôt dépend du harnais pour sa preuve.
3. L'option 1, plus une porte red minimale. Compromis : la discipline « test d'abord » reste, mais avec un reste de harnais.

## Decision

Chaque dépôt Go, Python ou Helm déclarera dans son `mise.toml` une tâche `mutation`, lancée par `check` et donc par la CI. Elle échoue quand le score de mutation passe sous `MUTATION_THRESHOLD` (100 par défaut). L'outil dépend de la pile :
- Go : gremlins, limité au code changé depuis `MUTATION_BASE` ;
- Python : pytest-gremlins, limité aux fichiers changés depuis `MUTATION_BASE`, avec le score lu par `jq` ;
- Helm : helm-mutation-test, limité aux changements depuis `MUTATION_BASE`.

Nous retirerons du harnais les commandes `gate red`, `green`, `characterize` et `replay`, le domaine `tdd`, l'agent `mutant-writer` et les fichiers `TESTS.md`. Nous garderons `gate coverage`. Un mutant équivalent s'exclut par la configuration de l'outil, jamais en baissant le seuil.

## Consequences

- Plus facile : un dépôt prouve ses tests sans Claude ni harnais (`mise run mutation`). Le harnais perd une large part de son code. Une tâche demande moins d'allers-retours d'agents.
- Plus difficile : l'ordre « test avant code » n'est plus vérifié. Python mute les fichiers changés en entier, anciens survivants compris, et pytest-gremlins a moins d'un an. gremlins ne détecte pas les équivalents, d'où des exclusions à consigner. helm-mutation-test est jeune et n'a qu'un mainteneur.
- À revoir si :
  - la tâche `mutation` d'un dépôt dépasse le temps de CI acceptable ;
  - les exclusions d'équivalents se multiplient au point de vider le seuil de son sens ;
  - un des trois outils n'est plus maintenu.
