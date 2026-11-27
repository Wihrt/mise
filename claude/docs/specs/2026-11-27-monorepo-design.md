# Monorepos : un dossier par service

Date : 2026-11-27. Statut : proposé. ADR : `docs/adr/0004-un-monorepo-est-un-monorepo-mise-un-mise-toml-par-service.md`.

## Intention

Dans un monorepo, le code est toujours dans un sous-dossier. Aujourd'hui, la chaîne ne voit pas ces sous-dossiers :
- `repo-stack` ne regarde que la racine. Il répond `stack=none`, donc pas de skill de pile, pas de test d'architecture, et l'étape de couverture est sautée. C'est ce qui arrive à ce dépôt avec `harness/`.
- `mutation:py` passe des chemins comptés depuis la racine à `--gremlin-targets` (remarque 1 de la revue finale de la pile « mutation outillée »).
- La porte de couverture, lancée à la racine avec un rapport LCOV écrit dans un service, ne juge pas les fichiers Python et sort en `PASS` (essai du 2026-11-27, ci-dessous).

On veut que toute la chaîne (détection de la pile, test d'architecture, couverture, mutation) fonctionne service par service, sans renommer de tâche à la main pour chaque service.

## Objectifs

- Un monorepo se déclare une fois, à la racine. Chaque service est ensuite traité comme un petit dépôt : les fragments de `repo-tooling` et les skills de pile s'y appliquent tels quels.
- La détection de la pile est déterministe et couvre chaque service.
- `mise run check` à la racine lance le `check` de chaque service : lint, tests, mutation.
- La porte de couverture juge chaque service.

## Non-objectifs

- Les autres formes de monorepo : code à la racine et services à côté, bibliothèques partagées (`go.work`, workspaces uv). Une bibliothèque listée comme service reçoit la réponse de `repo-stack` pour une bibliothèque (`stack=none`, demander à l'utilisateur), comme aujourd'hui à la racine.
- La détection par balayage des `go.mod` et `pyproject.toml` : un `pyproject.toml` qui ne sert qu'aux tests d'un chart serait pris pour un service.
- Un `COVERAGE.md` à la racine : chaque service a le sien.

## Anti-objectifs

- Un faux vert : une porte ou une tâche qui passe sans avoir jugé le code d'un service.
- Une écriture à la main par service au-delà de sa ligne dans `config_roots` et de son `mise.toml`.

## Contraintes (faits vérifiés le 2026-11-27, mise 2026.11.10)

- **mise gère les monorepos nativement.** `monorepo_root = true` et `[monorepo] config_roots = ["services/a", "services/b"]` dans le `mise.toml` racine. `mise run '//...:test'` lance la tâche `test` de chaque service, dans le dossier du service, avec ses propres `[tools]`. La découverte automatique et `experimental_monorepo_root` sont dépréciés : la liste `config_roots` est obligatoire. Doc : https://mise.jdx.dev/tasks/monorepo.html.
- **L'environnement descend.** Une tâche de service voit le `[env]` de la racine. `MUTATION_BASE` défini à la racine par `get_env(...)` vaut sa valeur par défaut, ou celle du shell (`MUTATION_BASE=foo mise run check` → `foo` dans les deux services).
- **La porte de couverture marche déjà par service.** `harness gate coverage --task test --repo <racine>/services/a --coverage .tdd/cover/go.txt` lance le `test` du service dans son dossier, ne liste que ses fichiers, et rapporte `UNCOVERED internal/domain/x.go:5-5` avec des chemins relatifs au service. Même chose en Python avec `py.lcov`. `COVERAGE.md` est lu dans le dossier passé à `--repo` (`harness/internal/app/gate/coverage.go:112`).
- **Faux vert de la porte à la racine.** `--repo <racine> --coverage services/b/.tdd/cover/py.lcov` : coverage.py écrit `SF:src/b/x.py`, qui ne correspond à aucun fichier suivi (`services/b/src/b/x.py`). Les fichiers Python sont écartés en silence : `NOT JUDGED 7 files`, aucune ligne `UNCOVERED` pour la fonction Python non testée. Les chemins Go, eux, sont résolus (`UNCOVERED services/a/...`).
- **pytest-gremlins résout `--gremlin-targets` depuis le `rootdir` de pytest** (`plugin.py`, `_resolve_target_paths`). Lancé dans `services/b`, `--gremlin-targets=src/pkg/mod.py` mute le fichier (« Zapped: 2 gremlins »).
- **gremlins** dans un sous-dossier : réglé par `diff.relative=true` (MUT-006).
- **Règle des scripts.** Le snippet bash de `repo-stack` décide un résultat (il teste des fichiers et nomme une pile) : la nouvelle détection, qui doit en plus lire un TOML, va en Go, avec tests.

## Approches

1. **Monorepo natif de mise, un `mise.toml` par service (retenue par l'utilisateur).** Chaque service est un petit dépôt ; la racine déclare `config_roots` et agrège. Les fragments s'appliquent sans changement ; les chemins Go et Python sont bons parce que les tâches tournent dans le service. Coût : la liste `config_roots` à tenir, des `[tools]` par service.
2. Un seul `mise.toml` racine, tâches suffixées (`test:a`, `mutation:go:a`). Chaque fragment renommé à la main pour chaque service ; `mutation:py` garde son défaut de chemins. Écartée : contraire au deuxième anti-objectif.
3. Détection par balayage au lieu de `config_roots`. Écartée : voir les non-objectifs, et mise exige la liste.

## Conception

### Déclaration

Le `mise.toml` racine d'un monorepo porte :
```toml
monorepo_root = true

[monorepo]
config_roots = ["services/a", "services/b"]
```
`config_roots` est la seule source de vérité de la liste des services, pour mise et pour le harnais.

### Détection : `harness local stack`

`repo-stack` appelle `harness local stack --repo <dépôt>` au lieu de son snippet bash. La commande :
- lit le `mise.toml` de la racine du dépôt (`mise.toml`, à défaut `.config/mise.toml`, `.mise.toml`) ;
- sans `monorepo_root = true` : une seule ligne pour la racine, `. stack=<pile> image=<yes|no>` ;
- avec `monorepo_root = true` : une ligne par entrée de `config_roots`, dans l'ordre de la liste, `<dossier> stack=<pile> image=<yes|no>` ;
- applique à chaque dossier la règle actuelle de `repo-stack` : `go.mod` et un dossier `cmd/` ou `internal/` → `stack-go-app` ; `pyproject.toml` et un dossier `src/` → `stack-py-app` ; les deux → deux lignes pour ce dossier ; ni l'un ni l'autre → `stack=none` ; `image=yes` quand le dossier a un `Dockerfile` ;
- sort en erreur (code 2, message sur la sortie d'erreur) quand `monorepo_root = true` sans `config_roots`, ou quand un dossier listé n'existe pas.

`repo-stack` suit le skill de pile nommé pour chaque dossier, dans ce dossier. Le tableau et la section « Limits » de `repo-stack` sont mis à jour.

### Outillage d'un monorepo (`repo-tooling`)

- Nouveau fragment `templates/monorepo.toml` pour la racine : `monorepo_root`, `[monorepo] config_roots`, `MUTATION_BASE` (déplacé depuis la partie commune de `mutation-go.toml`), et les tâches d'agrégation `test = "//...:test"`, `mutation = "//...:mutation"`, `check` qui dépend de `lint` (outils de dépôt : secrets, typos, prek) et de `//...:check`.
- Chaque service reçoit son `mise.toml` avec les fragments habituels, sans la partie commune `MUTATION_BASE`.
- Ce qui reste à la racine : prek, Renovate, gitleaks, typos, la CI. La CI lance `mise run check` à la racine, comme aujourd'hui.
- Un service Python lance `mutation:py` dans son dossier : `--gremlin-targets` reçoit alors des chemins relatifs au service. La garde `git diff` du fragment Python reçoit `diff.relative=true`, comme Go (MUT-006).

### Porte de couverture

- `stacked-tdd-mr` (étape 4c) lance la porte une fois par dossier que `harness local stack` nomme avec une pile : `--repo <dépôt>/<dossier>`. Chaque service a son `COVERAGE.md`.
- La porte refuse le faux vert : quand **aucun** fichier d'un rapport ne correspond à un fichier suivi du dépôt passé à `--repo`, elle sort en `ERROR: <rapport>: aucun fichier suivi (lancer la porte dans le dossier du service ?)` puis `FAIL`, code 2.

### `stacked-tdd-mr`

- Étape 3b : `harness local stack` remplace le snippet ; le socle (test d'architecture, `FAC-…`) se pose par service.
- Étape 4 : inchangée (`mise run check` à la racine lance tout).
- Étape 4c : une porte par service, comme ci-dessus.

## Choix de l'auteur (à relire)

- **Sortie de `harness local stack`** : une ligne par dossier et par pile, au format ci-dessus. `.` désigne la racine.
- **Faux vert de la porte** : l'erreur se déclenche quand aucun fichier du rapport n'est suivi, pas dès qu'un seul ne l'est pas. Des fichiers générés et ignorés par git peuvent légitimement apparaître dans un rapport Go.
- **Fichiers lus pour `monorepo_root`** : seulement le `mise.toml` racine (`mise.toml`, `.config/mise.toml`, `.mise.toml`, dans cet ordre), pas les `mise.local.toml`.
- **Ce dépôt** n'est pas converti en monorepo par cette pile : `harness/` reste un sous-dossier sans `config_roots`. La conversion serait une suite.

- **Chaque service a une tâche `check`**, y compris un chart (`lint`, `helm unittest`, `mutation:helm:<chart>`). `repo-tooling` le dit, et `//...:check` les lance toutes.
- **Chemins dans un service.** Les tâches tournent dans le dossier du service. `mutation:go` a déjà `diff.relative=true`. `mutation:py` le reçoit aussi, si bien que sa garde et `--gremlin-targets` voient des chemins relatifs au service, que pytest-gremlins résout depuis son `rootdir`, qui est le service.
- **Deux piles dans un dossier** (`go.mod` et `pyproject.toml`) : deux lignes, chacune avec la même valeur `image=`. Le `Dockerfile` est cherché dans le dossier lui-même, pas plus bas.
- **Entrées de `config_roots`** : chemins relatifs, sans `..`, sans doublon, et le dossier doit exister. Sinon, erreur (code 2). Les entrées imbriquées (`services` et `services/a`) sont refusées aussi.
- **Racine** : `lint` y porte les outils de dépôt (prek, gitleaks, typos). Le lint des services passe par `//...:check`.
- **Porte par dossier** : un seul lancement par dossier qui a au moins une pile, avec un `--coverage` par rapport (Go et Python). Un `COVERAGE.md` par dossier.
- **Rapport vide** : un rapport sans aucun fichier déclenche la même erreur qu'un rapport dont aucun fichier n'est suivi.
- **`repo-stack` dépend du binaire `harness`**, comme `stacked-tdd-mr` le fait déjà : le binaire est déployé par les dotfiles dans `~/.local/bin`. Pas de repli sur l'ancien snippet bash.
- **ADR 0004** : écrit avec cette spec, dans la même MR.

## Laissé au constructeur

- Vérifier si `mise install` lancé à la racine installe les `[tools]` des services (le fixture de MNR-004 et la CI en dépendent). Sinon, la CI lance `mise install` par service, dans une tâche de la racine.

- Le nom exact de la tâche d'agrégation de lint à la racine.
- La bibliothèque TOML Go : `harness/go.mod` n'en a plus depuis la pile « mutation outillée » (vérifié : aucune ligne `toml`). `github.com/BurntSushi/toml` ou `github.com/pelletier/go-toml/v2`, épinglée et suivie par Renovate.

## Critères d'acceptation

Frontière : la ligne de commande `harness`, et les tâches mise des fragments `repo-tooling` lancées sur des dépôts jetables.

### MNR-001 — `harness local stack` nomme la pile de chaque service d'un monorepo
- **Étant donné** un dépôt jetable dont le `mise.toml` racine porte `monorepo_root = true` et `config_roots = ["services/a", "services/b", "charts/c"]`, avec `services/a` une application Go (`go.mod`, `cmd/`, `Dockerfile`), `services/b` une application Python (`pyproject.toml`, `src/`), et `charts/c` un chart avec un `pyproject.toml` pour ses tests
- **Quand** on lance `harness local stack --repo <dépôt>`
- **Alors**
  - la sortie est exactement `services/a stack=stack-go-app image=yes`, `services/b stack=stack-py-app image=no`, `charts/c stack=none`, une par ligne, dans cet ordre
  - la commande sort en succès

### MNR-002 — `harness local stack` garde le comportement d'un dépôt simple
- **Étant donné** un dépôt jetable sans `monorepo_root`, avec `go.mod` et `internal/` à la racine
- **Quand** on lance `harness local stack --repo <dépôt>`
- **Alors**
  - la sortie est exactement `. stack=stack-go-app image=no`
  - avec `monorepo_root = true` sans `config_roots`, ou avec un dossier listé absent, la commande sort en erreur et le dit sur la sortie d'erreur

### MNR-003 — La porte de couverture refuse un rapport dont aucun fichier n'est suivi
- **Étant donné** un dépôt jetable avec un service Python `services/b`, dont le rapport `services/b/.tdd/cover/py.lcov` nomme `src/b/x.py`
- **Quand** on lance `harness gate coverage --repo <racine> --task test --coverage services/b/.tdd/cover/py.lcov`
- **Alors**
  - la porte sort en échec et nomme le rapport sur sa sortie
  - lancée avec `--repo <racine>/services/b --coverage .tdd/cover/py.lcov`, elle juge `src/b/x.py`

### MNR-004 — `mise run check` à la racine d'un monorepo lance la mutation de chaque service
- **Étant donné** un dépôt jetable construit avec `monorepo.toml` à la racine, un service Go `services/a` et un service Python `services/b` chacun avec son fragment de mutation, et une fonction ajoutée dans chaque service depuis `MUTATION_BASE`
- **Quand** on lance `mise run check` à la racine
- **Alors**
  - si les tests unitaires des deux services vérifient leurs résultats, la tâche sort en succès
  - si le test unitaire du service Python appelle sa fonction sans vérifier son résultat, la tâche sort en échec
  - si le test unitaire du service Go appelle sa fonction sans vérifier son résultat, la tâche sort en échec
