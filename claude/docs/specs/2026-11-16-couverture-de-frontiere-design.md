# Pas de code mort : la couverture des tests de frontière — spec

Date : 2026-11-16. Périmètre : le binaire `harness` (`harness/`, nouveau mode `gate coverage`), `claude/TESTS.md`, `claude/agents/implementer.md`, `claude/skills/{stacked-tdd-mr,repo-tooling,compose-dev,stack-go-app,stack-py-app}/`.

Décision d'architecture : [ADR 2](../../../docs/adr/0002-le-code-mort-se-prouve-par-la-couverture-des-tests-de-frontiere.md).

## Objectif

Demande du 16/11 : le harnais garantit trois choses sur une application Go ou Python. L'architecture hexagonale, par des tests d'architecture. Des tests d'acceptation qui passent à chaque fois, pour pouvoir refactorer. Et « idéalement, tous les chemins de code doivent être testés, qu'on n'ait pas de code mort ».

Les deux premières sont en place (`2026-11-13-filets-de-refactoring-design.md`). La troisième n'a plus rien derrière elle : la preuve « code minimal » a été retirée le 16/11 parce qu'elle relançait la suite une fois par bloc ajouté (34 minutes mesurées le 13/11), et la même spec interdit tout contrôle dont la durée grandit avec la quantité de code (lignes 61-64).

Cette spec la remplace par une mesure de couverture : une seule exécution de la suite, lue par la porte.

Faits établis le 16/11 :

- Les tests d'acceptation n'appellent jamais le code dans leur propre processus : sous-processus pour un outil en ligne de commande (`claude/agents/acceptance-writer.md:49`), conteneur lancé par compose pour une API (`acceptance-writer.md:52`) et pour un front web (`acceptance-writer.md:53`).
- Essais jetables, rejoués : un service Go (1.27) ou Python (3.14, coverage.py 7.17.3) construit avec la mesure, lancé par compose et appelé seulement de l'extérieur, rend sa mesure à l'arrêt s'il s'arrête proprement sur SIGTERM. Un service Go qui ne traite pas SIGTERM sort à 0 %. Go mesure des blocs d'instructions, Python des lignes et des branches.
- Mesure sur le harnais : ses tests d'acceptation Go exécutent 93,0 % des instructions de `internal/app/gate`. Sur 107 blocs non exécutés hors code testé ailleurs, 39 ne font que remonter une erreur.

## Décisions prises

| Sujet | Décision | Par |
|---|---|---|
| Ce qui est superflu | Le code qu'aucun test d'acceptation n'atteint. | utilisateur, 13/11 |
| Qui écrit le rapport | La tâche de test, pendant son exécution normale ; la porte le lit. | utilisateur, 13/11 (décision 4) |
| Quels tests comptent | Seulement ceux qui passent par la frontière, service compris. L'image testée embarque la mesure. | utilisateur, 16/11 (décision 8) |
| Où vit le contrôle | Dans la porte. | utilisateur, 16/11 (décision 9) |
| Sur quel code | Tout le code de l'application, pas seulement les lignes ajoutées. | utilisateur, 16/11 (décision 10) |
| Code hors d'atteinte | La porte écarte d'elle-même les blocs Go qui ne font que remonter une erreur, en imprimant combien ; elle reste stricte pour le reste. | utilisateur, 16/11 (décision 11) |
| Fichiers sans mesure possible | Ils ne sont pas jugés et la porte le dit. | utilisateur, 13/11 (décision 5) |
| Forme | Un mode à part, `harness gate coverage`, et non une option de `green` : la mesure n'est complète que si toutes les suites tournent, alors que `green` ne lance que la tâche du critère jugé. | choix de conception |
| Fréquence | Une fois par tâche du plan, après la porte verte, et une fois en fin de pile. | choix de conception |
| Exceptions | Dans un fichier du dépôt, `COVERAGE.md`, à la racine donnée par `--repo`. L'implementer ne le modifie pas ; l'orchestrateur y écrit après une décision tracée. | choix de conception |
| Fichier Go sans fonction | Un fichier Go qui ne déclare aucune fonction (interfaces, types, constantes) n'a rien à exécuter : il n'est pas jugé. Cas réel : `harness/internal/app/gate/ports.go`. | choix de conception |
| Ancrage d'une exception | Par le texte de la ligne, pas par son numéro : l'exception survit à un déplacement du code. | choix de conception |
| Exception périmée | Une exception qui ne correspond plus à rien fait échouer la porte : la liste se vide d'elle-même. | choix de conception |
| `# pragma: no cover` | Refusé dans le code Python de production. Une exception s'écrit dans `COVERAGE.md`, jamais dans le code. | choix de conception |
| Fichiers d'un autre langage | Décision 5 appliquée à tout le dépôt : une seule ligne en donne le nombre, pour ne pas noyer la sortie. Un fichier Go ou Python que nul rapport ne mesure est, lui, un échec. | choix de conception |
| Dette de départ | `--baseline` écrit dans `COVERAGE.md` ce qui n'est pas couvert aujourd'hui, avec la raison « dette initiale ». | choix de conception (promis avec la décision 10) |
| Dépôt du harnais | Hors périmètre : ses hooks sont testés par bats avec un binaire non instrumenté. | choix de conception |
| Construction | En TDD à la main sur ce dépôt, sans `harness gate`, comme D9 de `2026-11-13-filets-de-refactoring-design.md`. | décision antérieure |
| Identifiants | Nouvelle série `COV`. | choix de conception |

## Conception

### Le mode `harness gate coverage`

```
harness gate coverage --task <tâche> --repo <racine> --coverage <rapport> [--coverage <rapport> ...] [--baseline] [--timeout <s>]
```

Le mode ne lit ni `TESTS.md` ni slug. Il fait, dans l'ordre :

1. supprimer chaque rapport de `--coverage` resté d'une exécution précédente ;
2. lancer `mise run <tâche>` une fois ; si la tâche échoue, imprimer `TASK FAILED <tâche>` et finir en `FAIL` : la couverture d'une suite rouge ne dit rien ;
3. lire chaque rapport ; un rapport non écrit donne `COVERAGE missing: <tâche> n'a pas écrit <chemin>` et `FAIL` ; un rapport lu donne `COVERAGE <chemin>: <n> files` ;
4. réunir les rapports : un bloc, une ligne ou une branche est couvert dès qu'un rapport le dit exécuté ; seul compte « exécuté ou non », donc des profils Go de modes différents (`set`, `count`, `atomic`) se réunissent ;
5. juger, puis finir par `PASS` (code 0) ou `FAIL` (code 1). Une erreur d'usage ou d'environnement suit la règle des autres modes : `ERROR:`, `FAIL`, code 2.

Deux formats de rapport, reconnus à leur contenu :

- **profil Go** en texte (première ligne `mode:`), tel que l'écrit `go tool covdata textfmt`. Chaque ligne nomme un fichier par son chemin de module ; la porte le ramène à un fichier du dépôt par la ligne `module` des `go.mod` suivis par git, le module au chemin le plus long l'emportant quand plusieurs conviennent.
- **LCOV** (lignes `SF:`, `DA:`, `BRDA:`), tel que l'écrit `coverage lcov`, avec des chemins relatifs à la racine du dépôt. Dans une ligne `BRDA:`, le compteur `-` vaut « jamais prise ».

### Ce que la porte juge

Le code de production en Go et en Python : les fichiers `.go` et `.py` suivis par git, hors chemins de test (`tdd.IsTestPath` : `_test.go`, dossiers de tests). Un fichier Go qui ne déclare aucune fonction n'est pas jugé.

| Constat | Ligne imprimée | Effet |
|---|---|---|
| Bloc Go jamais exécuté | `UNCOVERED <fichier>:<début>-<fin>` | `FAIL` |
| Lignes Python jamais exécutées, groupées quand elles se suivent | `UNCOVERED <fichier>:<début>-<fin>` | `FAIL` |
| Branche Python jamais prise, sur une ligne exécutée | `UNCOVERED BRANCH <fichier>:<ligne>` | `FAIL` |
| Fichier `.go` ou `.py` de production absent de tous les rapports | `NOT MEASURED <fichier>` | `FAIL` |
| `# pragma: no cover` ou `# pragma: no branch` dans un fichier Python de production | `PRAGMA <fichier>:<ligne>` | `FAIL` |
| Fichiers suivis dont l'extension n'est ni `.go` ni `.py` | `NOT JUDGED <n> files: neither Go nor Python`, une seule ligne | aucun |

Précisions :

- **Bloc Go** : `<début>` et `<fin>` sont les lignes du profil. Le profil fait commencer un bloc à sa première instruction, pas à l'accolade ni au `if` : sur le harnais, le corps du `if err != nil {` de `characterize.go:96` est le bloc `97.3,98.1`, et le `default:` de `runner.go:31` le bloc `32.3,32.19`. La « première ligne » d'un bloc est donc celle de sa première instruction.
- **Bloc Go sans instruction** (corps vide ; le profil lui donne 0 instruction, 3 cas sur le harnais) : pas jugé.
- **Fichier Python vide** (`__init__.py`) : `coverage lcov` lui écrit une ligne `SF:` sans ligne `DA:` (vérifié le 16/11) ; il est donc mesuré, sans constat.
- **Lignes Python** : des lignes jamais exécutées se groupent quand aucune ligne exécutée ne les sépare dans le rapport ; les lignes vides et les commentaires, que le rapport ne mesure pas, ne coupent pas le groupe.
- **Branche** : une branche dont la ligne n'a jamais été exécutée n'est pas répétée, la ligne `UNCOVERED` la couvre.
- **Fichier Go exclu de la construction** (contrainte `//go:build`, suffixe `_windows.go`) : absent du profil, donc `NOT MEASURED`. Il se règle par une exception `fichier entier`.

Finesse, à dire telle quelle dans la compétence : en Go, la mesure porte sur des blocs d'instructions ; un `if` sans `else` n'a pas de bloc pour le cas faux. « Tous les chemins » veut dire blocs en Go, lignes et branches en Python.

### Blocs Go qui ne font que remonter une erreur

Un bloc Go jamais exécuté est écarté d'office quand il est le corps entier d'un `if` dont la condition compare à `nil`, par `!=`, un identifiant nommé `err` ou dont le nom finit par `err` ou `Err`, et que ce corps est une seule instruction, un `return`. La forme `if err := f(); err != nil` et un `else if` comptent ; un corps qui fait autre chose avant de rendre la main (journal, nettoyage) ne compte pas, ni le `else`.

```go
if err != nil {
    return nil, fmt.Errorf("lecture: %w", err)   // écarté
}
if rc != 0 {
    return errors.New("git a échoué")            // jugé : c'est une décision, pas une remontée
}
```

La porte imprime une ligne `ERROR RETURNS IGNORED <n>`, même quand `n` vaut 0 : `n` est le nombre de blocs jamais exécutés qu'elle a écartés ainsi. Un bloc écarté ne passe pas par `COVERAGE.md` : une entrée écrite pour lui n'accepte rien, donc elle est périmée. Elle lit pour cela le code Go du dépôt avec l'analyseur de la bibliothèque standard : aucune dépendance nouvelle. En Python, rien d'équivalent : la règle reste stricte.

### Exceptions : `COVERAGE.md`

À la racine donnée par `--repo`. Une section par fichier, une entrée par exception, la raison obligatoire :

```markdown
## internal/adapters/git/runner.go
- `return 127, text` : git introuvable, non provoquable depuis la frontière

## tools/gen.go
- fichier entier : générateur lancé à la main, hors produit
```

- Une entrée `` `<texte>` `` accepte, dans ce fichier, tout constat `UNCOVERED` ou `UNCOVERED BRANCH` dont la première ligne, espaces de début et de fin retirés, est ce texte. Elle accepte toutes les occurrences : deux `return nil, err` jamais exécutés dans un fichier se règlent par une entrée.
- Le texte est une portion de code Markdown : il est entouré d'autant d'accents graves qu'il en faut pour dépasser la plus longue suite d'accents graves qu'il contient, avec une espace de chaque côté quand il commence ou finit par un accent grave.
- Une entrée `fichier entier` accepte `NOT MEASURED` et tous les constats de ce fichier.
- Un constat accepté imprime `ACCEPTED <fichier>:<début>-<fin>: <raison>` au lieu de `UNCOVERED`, `ACCEPTED <fichier>: <raison>` au lieu de `NOT MEASURED`, et ne fait pas échouer.
- Une entrée qui n'accepte plus rien imprime `STALE EXCEPTION <fichier>: <texte>` et fait échouer.
- `PRAGMA` ne s'accepte pas.

`--baseline` : après le jugement, la porte ajoute à `COVERAGE.md`, sans toucher aux entrées déjà là, la raison `dette initiale` pour ce qui n'est pas accepté : une entrée par texte distinct pour les constats `UNCOVERED` et `UNCOVERED BRANCH` d'un fichier, une entrée `fichier entier` par `NOT MEASURED`. Ces constats ne font alors pas échouer. Tout le reste garde son effet : `PRAGMA`, `STALE EXCEPTION`, `COVERAGE missing` et `TASK FAILED` finissent en `FAIL`, et dans les deux derniers cas rien n'est écrit. C'est le geste d'adoption d'un dépôt existant, fait une fois par l'orchestrateur après une décision tracée.

### Ce que le dépôt applicatif doit fournir

Écrit dans les compétences, sans contrôle de la porte autre que le jugement lui-même.

- **La mesure tourne dans le produit, jamais dans le lanceur de tests.** Le binaire ou le conteneur testé est instrumenté ; `go test` et `pytest` ne le sont pas. Sinon les tests unitaires compteraient.
- **Go** : le produit est construit avec `-cover -coverpkg=./...` ; `GOCOVERDIR` pointe un dossier sous `.tdd/` ; la tâche `test` finit par `go tool covdata textfmt`.
- **Python** : coverage.py est démarré dans le produit par `COVERAGE_PROCESS_START`, avec `branch`, `parallel`, `relative_files` et `source` ; la tâche `test` finit par `coverage combine` et `coverage lcov`.
- **Conteneur** (`compose-dev`) : le Dockerfile de production prend un argument de construction qui active la mesure, vide par défaut ; les tâches de test le passent, montent le dossier des mesures et le rendent inscriptible par l'utilisateur du conteneur. L'image publiée et `mise run dev` ne sont pas instrumentées. C'est l'exception écrite à la règle « même image qu'en production » : même Dockerfile, même cible, un argument de plus.
- **Arrêt propre** : la mesure n'est écrite qu'à un arrêt normal. Le socle l'exige déjà (FAC-004).
- **`test`** lance toutes les suites, puis convertit : c'est la tâche que reçoit `gate coverage`.

### Ce qui change dans la procédure

- `stacked-tdd-mr` : après la porte verte de chaque tâche, quand le dépôt a une compétence de stack, `harness gate coverage --task test`. `UNCOVERED` : l'implementer retire le code, ou le signale par `UNTESTED` et l'orchestrateur tranche comme aujourd'hui (un critère de plus, ou une entrée dans `COVERAGE.md` avec une décision tracée). Même commande en fin de pile.
- `implementer.md` : tout bloc doit être atteint par un test d'acceptation ; ne modifie pas `COVERAGE.md` ; n'écrit pas de `# pragma: no cover`.
- `repo-tooling`, `compose-dev`, `stack-go-app`, `stack-py-app` : les règles de la section précédente, avec leurs gabarits.

Vérification des gabarits : deux applications d'essai hors dépôt, une Go et une Python, service en conteneur et outil en ligne de commande, dont la tâche `test` écrit des rapports que `gate coverage` accepte (`PASS` avec tout couvert, `UNCOVERED` sur une route jamais appelée).

### Approches

- **Tests comptés, retenu** : la frontière seule. Une ligne n'existe que si un critère en a besoin. **Non retenu** : tous les tests plus un outil statique de code mort (un test unitaire maintient en vie du code qu'aucune entrée réelle n'atteint) ; la frontière pour les seuls outils en ligne de commande (les services, cas principal, ne seraient pas couverts).
- **Lieu du contrôle, retenu** : la porte. Aucune exécution de plus que la suite, exceptions que l'auteur du code ne s'accorde pas. **Non retenu** : un seuil dans la tâche de test de chaque dépôt (deux mécanismes, exceptions par simple commentaire).
- **Code hors d'atteinte, retenu** : remontées d'erreur Go écartées d'office, le reste strict. **Non retenu** : tout strict (chaque remontée devient une exception, la liste grossit avec le code) ; un seuil (du code mort se loge dans la part tolérée).
- **Forme, retenu** : un mode à part. **Non retenu** : une option de `green` (elle ne lance qu'une suite, la mesure serait partielle) ; un outil hors du harnais (les exceptions et la procédure vivent déjà dans la porte).

### Limites connues, hors périmètre

- Le dépôt du harnais lui-même, et toute bibliothèque (sa frontière est un appel dans le processus du test).
- Un serveur Python à plusieurs processus (gunicorn, workers uvicorn) : non essayé.
- La mise en conformité des dépôts existants, au-delà de `--baseline`.
- Un langage autre que Go et Python.
- Une exception vaut pour toutes les lignes du fichier qui portent son texte : une nouvelle ligne identique, jamais exécutée, est acceptée sans décision. Ancrer plus finement (par fonction) attendra qu'on le constate.
- Empêcher par la porte qu'un implementer modifie `COVERAGE.md` : la règle est écrite, et chaque exécution imprime toutes les lignes `ACCEPTED`. On durcira si un agent passe outre.

## Critères d'acceptation

Frontière de tous les critères : la ligne de commande `harness gate coverage` sur un dépôt git jetable dont la tâche de test écrit les rapports donnés ; sortie standard et code de sortie. Les lignes citées (`UNCOVERED`, `UNCOVERED BRANCH`, `NOT MEASURED`, `NOT JUDGED`, `PRAGMA`, `ACCEPTED`, `STALE EXCEPTION`, `ERROR RETURNS IGNORED`, `COVERAGE missing`, `TASK FAILED`, `PASS`, `FAIL`) sont le contrat de la porte : `stacked-tdd-mr` les lit telles quelles.

### COV-001 — Un bloc Go jamais exécuté fait échouer la porte
- **Étant donné** un dépôt avec un `go.mod`, un fichier Go de production, et une tâche de test qui écrit un profil Go où un bloc de ce fichier a un compteur à 0 et les autres un compteur positif
- **Quand** `harness gate coverage` est lancé avec ce profil
- **Alors**
  - la sortie contient `UNCOVERED` avec le chemin du fichier dans le dépôt et les lignes du bloc, et se termine par `FAIL`
  - quand tous les blocs du profil ont un compteur positif, la sortie ne contient aucune ligne `UNCOVERED` et se termine par `PASS`
  - dans ce second cas, la sortie contient `ERROR RETURNS IGNORED 0`
  - quand le seul bloc au compteur 0 est un bloc sans instruction, la sortie ne contient aucune ligne `UNCOVERED` et se termine par `PASS`
- **Cas réel** : essai du 16/11, route jamais appelée d'un service en conteneur : `example.test/spike/internal/app/app.go:6.54,6.82 1 0` ; bloc sans instruction sur le harnais : 3 lignes du profil du 16/11 portent `0` instruction

### COV-002 — Une ligne ou une branche Python jamais exécutée fait échouer la porte
- **Étant donné** un fichier Python de production et une tâche de test qui écrit un rapport LCOV où des lignes qui se suivent ont un compteur à 0, la première portant des branches au compteur `-`, et où une ligne exécutée porte une branche jamais prise
- **Quand** `harness gate coverage` est lancé avec ce rapport
- **Alors**
  - la sortie contient une seule ligne `UNCOVERED` pour ces lignes, avec le fichier et leur intervalle
  - elle contient `UNCOVERED BRANCH` avec le fichier et la ligne exécutée qui porte la branche jamais prise
  - elle ne contient aucune ligne `UNCOVERED BRANCH` pour la ligne jamais exécutée
  - elle se termine par `FAIL`
- **Cas réel** : essai du 16/11 : `DA:19,0`, `DA:20,0` (route jamais appelée), `BRDA:19,0,jump to line 20,-` sur cette ligne 19, et `BRDA:16,0,jump to line 19,0` sur une ligne `DA:16,1`

### COV-003 — Les rapports de plusieurs suites se complètent
- **Étant donné** deux rapports et trois blocs : le premier exécuté dans le premier rapport seulement, le deuxième dans le second seulement, le troisième à 0 dans les deux
- **Quand** `harness gate coverage` est lancé avec `--coverage` donné deux fois
- **Alors**
  - la sortie ne contient aucune ligne `UNCOVERED` pour le premier bloc
  - elle n'en contient aucune pour le deuxième
  - elle contient `UNCOVERED` pour le troisième et se termine par `FAIL`
- **Cas réel** : mesure du 16/11 sur le harnais : 478 blocs à 0 sous les seuls tests Go, dans du code que les tests bats exécutent

### COV-004 — Une remontée d'erreur Go est écartée et comptée
- **Étant donné** un fichier Go avec deux blocs jamais exécutés : le corps d'un `if err != nil` qui ne contient qu'un `return`, et le corps d'un `if rc != 0` qui ne contient qu'un `return`
- **Quand** `harness gate coverage` est lancé
- **Alors**
  - la sortie ne contient aucune ligne `UNCOVERED` pour le premier bloc, et contient `ERROR RETURNS IGNORED 1`
  - elle contient `UNCOVERED` pour le second et se termine par `FAIL`
- **Cas réel** : harnais, 16/11 : `internal/app/gate/characterize.go:97-98` (`return false, failure{err.Error()}` sous `if err != nil`) et `internal/adapters/git/runner.go:40-41` (`return "", errors.New(...)` sous un autre test)

### COV-005 — Une exception écrite est acceptée, une exception périmée fait échouer
- **Étant donné** un bloc jamais exécuté, et `COVERAGE.md` qui porte, pour son fichier, une entrée avec le texte de sa première ligne et une raison
- **Quand** `harness gate coverage` est lancé
- **Alors**
  - la sortie contient `ACCEPTED` avec le fichier, les lignes du bloc et la raison, aucune ligne `UNCOVERED`, et se termine par `PASS`
  - quand le bloc est exécuté et que l'entrée reste, la sortie contient `STALE EXCEPTION` avec le fichier et le texte, et se termine par `FAIL`
- **Cas réel** : harnais, 16/11 : `internal/adapters/git/runner.go:32`, `return 127, text`, atteint seulement quand git est introuvable

### COV-006 — Un fichier Go ou Python que nul rapport ne mesure est signalé
- **Étant donné** un dépôt qui suit un fichier Python de production absent de tout rapport, un fichier de test Python absent lui aussi, un fichier Go de production absent lui aussi et qui ne déclare aucune fonction, et un fichier YAML
- **Quand** `harness gate coverage` est lancé
- **Alors**
  - la sortie contient `NOT MEASURED` avec le fichier de production et se termine par `FAIL`
  - elle ne contient aucune ligne `NOT MEASURED` pour le fichier de test ni pour le fichier YAML
  - elle n'en contient aucune pour le fichier Go sans fonction
  - elle contient une ligne `NOT JUDGED` qui compte le fichier YAML
  - quand `COVERAGE.md` porte une entrée `fichier entier` pour le fichier de production, la sortie contient `ACCEPTED` pour lui et se termine par `PASS`
- **Cas réel** : essai du 16/11 : un module Python jamais importé apparaît à 0 % avec le réglage `source`, et disparaît du rapport sans lui ; fichier Go sans fonction : `harness/internal/app/gate/ports.go`, absent du profil du 16/11

### COV-007 — Un rapport non écrit ou une suite en échec ne passent pas
- **Étant donné** un rapport resté d'une exécution précédente, où tout est couvert
- **Quand** `harness gate coverage` est lancé avec une tâche qui n'écrit pas ce rapport, puis avec une tâche qui l'écrit et échoue
- **Alors**
  - dans le premier cas, la sortie contient `COVERAGE missing` avec le chemin du rapport et se termine par `FAIL`
  - dans le second, elle contient `TASK FAILED` et se termine par `FAIL`
- **Cas réel** : essai du 16/11 : dossier de mesure non inscriptible par l'utilisateur du conteneur, rien n'est écrit et le produit sort quand même en code 0

### COV-008 — `# pragma: no cover` est refusé
- **Étant donné** un fichier Python de production dont une ligne porte `# pragma: no cover` et une autre `# pragma: no branch`, et un rapport où tout ce qui est mesuré est couvert
- **Quand** `harness gate coverage` est lancé
- **Alors**
  - la sortie contient `PRAGMA` avec le fichier et la ligne du `no cover`, et se termine par `FAIL`
  - elle contient `PRAGMA` avec le fichier et la ligne du `no branch`
- **Cas réel** : aucun constaté ; règle écrite par cette spec (décision 9 : une exception ne se pose pas par commentaire)

### COV-009 — `--baseline` écrit la dette de départ
- **Étant donné** un dépôt avec deux blocs jamais exécutés et aucun `COVERAGE.md`
- **Quand** `harness gate coverage --baseline` est lancé, puis `harness gate coverage` sans l'option
- **Alors**
  - après le premier lancement, `COVERAGE.md` porte, sous le fichier de chaque bloc, une entrée avec le texte de sa première ligne et la raison `dette initiale` ; ce lancement se termine par `PASS`
  - le second lancement imprime `ACCEPTED` pour chacun, aucune ligne `UNCOVERED`, et se termine par `PASS`
- **Cas réel** : mesure du 16/11 sur le harnais : 107 blocs jamais exécutés dans du code déjà en place
