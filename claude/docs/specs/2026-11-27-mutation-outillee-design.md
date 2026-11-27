# Mutation outillée : les dépôts prouvent leurs tests avec gremlins, pytest-gremlins et helm-mutation-test

Statut : proposé, 2026-11-27 (révisé le même jour : la mutation mesure les tests unitaires). ADR : `docs/adr/0003-la-preuve-des-tests-est-une-mutation-outillee-dans-chaque-depot.md`.

## Intention

Aujourd'hui, c'est le harnais qui prouve la valeur d'un test. `harness gate red` montre le test d'acceptation rouge. Puis l'agent `mutant-writer` écrit un mutant par clause de critère, et `harness gate green --proofs` vérifie que chaque mutant est tué (`claude/skills/stacked-tdd-mr/SKILL.md:11-14`). Cette chaîne est lente et dépend de l'IA à chaque maillon : un agent écrit les mutants, l'orchestrateur tranche `SURVIVED`, `REDUNDANT`, `NO-MUTANT`.

L'utilisateur veut réduire cette dépendance. La preuve devient le travail d'outils de mutation standard, déclarés dans chaque dépôt et lancés par `mise`, en local comme en CI. L'IA reçoit une boîte à outils et s'en sert ; elle n'est plus elle-même la preuve. N'importe qui peut lancer `mise run mutation` sans Claude.

Deuxième demande, liée : un script reste en bash tant qu'il séquence des commandes. Dès qu'il décide d'un résultat par lui-même, il est écrit en Python ou en Go, avec des tests.

## Objectifs

- Chaque dépôt Go, Python ou Helm a une tâche `mutation`. Elle échoue dès qu'un mutant du code changé survit ou n'est atteint par aucun test. L'outil dépend de la pile :
  - Go : gremlins ;
  - Python : pytest-gremlins ;
  - Helm : helm-mutation-test (les charts ont déjà leurs suites helm-unittest, exigées par les skills Helm).
- En Go et en Python, la mutation mesure les **tests unitaires**, qui tournent dans le processus. Les tests d'acceptation restent à la frontière, et la porte de couverture prouve qu'ils atteignent tout le code. Les tests unitaires deviennent obligatoires (choix de l'utilisateur, option 1 du 2026-11-27).
- `check`, donc la CI, lance `mutation`. La CI verte est le verdict de la merge request.
- Le harnais perd la preuve TDD : `harness gate red`, `green`, `characterize` et `replay`, le domaine `tdd`, l'agent `mutant-writer` et les fichiers `TESTS.md` disparaissent.
- `stacked-tdd-mr` devient plus court : critères, tests d'acceptation, code et tests unitaires, `mise run check`, couverture, revue, MR.
- La règle des scripts est écrite dans `claude/CLAUDE.md` et dans `repo-tooling`.

## Non-objectifs

- La porte de couverture `harness gate coverage` reste telle quelle (choix de l'utilisateur : « couverture gardée »). Elle est déjà déterministe, et ne lit ni `TESTS.md` ni slug (`SKILL.md:15`). Seule sa dépendance à `tdd.IsTestPath` (`harness/internal/app/gate/coverage.go:62`) est déplacée.
- Pas de mutation pour OpenTofu, Kyverno, Ansible ni les autres livrables du catalogue : aucun outil n'a été retenu, ils gardent leurs tests natifs.
- Pas de migration, dans cette pile, des scripts bash existants qui enfreignent la nouvelle règle. Ils sont inventoriés dans la section « Suites ».
- Pas de nouveau binaire ni de nouvelle commande `harness` pour piloter les outils (option 2 écartée).

## Anti-objectifs

- Un dépôt dont la tâche `mutation` ne tourne que sur le poste de l'utilisateur (outil installé globalement, plugin Helm installé à la main, `harness` requis). C'est un échec, même si la CI passe ailleurs.
- Un survivant que l'orchestrateur laisse passer pour finir une tâche. Seul un mutant équivalent peut être exclu, par la configuration de l'outil, et l'exclusion est consignée.
- Une tâche `mutation` qui sort en succès sans avoir testé le code changé (faux vert).
- Réintroduire un agent qui juge les survivants à la place de l'outil.

## Contraintes (faits vérifiés le 2026-11-27)

- Aucun des trois outils n'est au registre mise ni au registre aqua (`mise registry`, 1077 entrées ; `aquaproj/aqua-registry/pkgs/...` en 404).
- **gremlins** v0.6.0 (2025-12-06, Apache-2.0)
  - `gremlins unleash --diff <ref>` ne mute que le code changé. `-o file.json` écrit `test_efficacy`, `mutations_coverage`, `mutants_killed`, `mutants_lived`, `mutants_not_covered`, …
  - Les seuils natifs ne fonctionnent pas. La doc annonce `--threshold-efficacy` (code 10) et `--threshold-mcover` (code 11), mais **l'essai sort en 0** avec `--threshold-mcover 90` pour une couverture de 50 %. La tâche ne s'appuie donc pas sur eux.
  - gremlins ne mute que le code couvert par les tests **du processus** `go test`. Sur un module dont le seul test compile et lance le binaire, l'essai donne `Killed: 0, Lived: 0, Not covered: 2` et sort en 0.
  - Pas de statut « équivalent ».
  - Archive `github:` avec le binaire `gremlins` à la racine (vérifié par extraction).
  - Source : https://github.com/go-gremlins/gremlins/blob/main/docs/docs/usage/commands/unleash/index.md
- **pytest-gremlins** 1.15.0 (PyPI 2026-11-26, MIT, projet créé en 12/2025)
  - Plugin pytest (`pytest --gremlins`), parallèle, avec un cache par empreinte de contenu (`--gremlin-cache`).
  - Pas de seuil : la doc le dit (`docs/cookbook/ci-integration.md:12`) et propose `jq` sur le rapport.
  - Pas de mode diff : `--gremlin-targets` reçoit une liste de fichiers, donc le code changé se limite au fichier.
  - Les mutants équivalents sont « graciés » dans le code, avec un plafond fixé par `--gremlin-max-pardons-pct`.
  - Essai jetable :
    - avec 2 fonctions et 1 test sans assertion, on obtient 4 mutants dont 2 survivants, et **le code de sortie est 0** ;
    - le rapport est `coverage/gremlins/gremlins.json`, avec `summary = {total, zapped, survived, timeout, error, pardoned, percentage}` ;
    - `--gremlin-targets=` vide mute tout le paquet ;
    - une fonction sans aucun test compte en `survived`.
  - Les mutants sont injectés dans le processus pytest. Un test qui lance un conteneur ou un sous-processus ne les voit pas : c'est déduit du mécanisme, mais pas essayé.
  - Source : https://github.com/mikelane/pytest-gremlins
- **Écartés pour Python** (choix de l'utilisateur : pytest-gremlins) :
  - mutmut 3.8.1 : ni seuil ni diff, il mute tout le paquet ;
  - cosmic-ray 8.9.0 : seuil et filtre git natifs, mais il relance la suite à chaque mutant et ignore `--fail-over 0`.
- **helm-mutation-test** v1.2.0 (2026-10-22, un seul mainteneur, Apache-2.0)
  - Il embarque helm-unittest.
  - Options `--since <ref>` et `--threshold N` (code 2 si les suites échouent déjà). `--since` travaille au fichier : un template touché est muté en entier (constaté le 2026-11-27 pendant la tâche 3, « 2 files inspected », survivants sur les lignes non modifiées).
  - Les mutants équivalents sont détectés et exclus. `noCoverage` (un template qu'aucune suite ne rend) est hors du score.
  - Le rapport est `mutation-report.json`, avec `tally{killed, survived, noCoverage, invalid, equivalent, timeout, error}`.
  - Un changement de `values.yaml`, d'un partiel `_*.tpl` ou de `Chart.yaml` déclenche un run complet.
  - Le binaire de release fonctionne seul, sans Helm.
  - Source : https://github.com/mmpyro/helm-mutation-test/blob/main/docs/cli-reference.md
- Règle `repo-tooling` : tout outil est déclaré dans le `mise.toml` du dépôt, en version exacte, et les tâches sont inline.

## Approches considérées

1. **Outils dans chaque dépôt, la porte disparaît** (retenue par l'utilisateur). Une tâche `mutation` par pile, avec le verdict en CI. `gate red/green/...`, `mutant-writer` et `TESTS.md` sont supprimés. Compromis : plus rien ne vérifie que le test a été écrit avant le code.
2. **Le harnais enveloppe les outils.** `harness gate` lance les outils et lit leurs rapports. Compromis : on garde du code maison, et un dépôt dépend encore du harnais pour sa preuve.
3. **L'option 1, plus une porte red minimale.** Compromis : on garde la discipline « test d'abord », au prix d'un reste de harnais et de l'agent qui l'orchestre.

Sur ce que mesure la mutation en Go et en Python, la question s'est posée après l'essai gremlins :
1. **La mutation porte sur les tests unitaires** (retenue).
2. Les tests d'acceptation appellent le code dans le processus : c'est l'abandon de la frontière.
3. La mutation se limite à Helm : on garde le `mutant-writer`.

## Conception

### Boîte à outils dans `repo-tooling`

Trois fragments s'ajoutent à `claude/skills/repo-tooling/templates/` : `mutation-go.toml`, `mutation-py.toml` et `mutation-helm.toml`. Ce sont des TOML valides, à fusionner dans le `mise.toml` du dépôt selon sa pile. Le skill dit quand les prendre. Le verdict ne dépend jamais du code de sortie de l'outil, mais d'un `jq` sur son rapport : **aucun survivant et aucun mutant non atteint par un test**.

```toml
# commun
[tools]
jq = "<version exacte>"
[env]
MUTATION_BASE = "origin/main"   # la CI le remplace par la branche cible de la MR
[tasks.mutation]
depends = ["mutation:*"]
[tasks.check]
depends = ["lint", "test", "mutation"]

# mutation-go.toml
[tools]
"github:go-gremlins/gremlins" = "0.6.0"
[tasks."mutation:go"]
run = '''
set -eu
mkdir -p .tdd
gremlins unleash --diff "$MUTATION_BASE" -o .tdd/mutation-go.json
jq -e '.mutants_lived == 0 and .mutants_not_covered == 0' .tdd/mutation-go.json
'''

# mutation-py.toml (pytest-gremlins dans le groupe dev de pyproject.toml)
[tasks."mutation:py"]
run = '''
set -eu
targets=$(git diff --name-only --diff-filter=AMR "$MUTATION_BASE" -- '<paquet>/*.py' | paste -sd, -)
test -n "$targets" || exit 0
uv run pytest -q tests/unit --gremlins --gremlin-cache --gremlin-targets="$targets" --gremlin-report=json
jq -e '.summary.survived == 0' coverage/gremlins/gremlins.json
'''

# mutation-helm.toml (une tâche par chart)
[tools]
"github:mmpyro/helm-mutation-test" = "1.2.0"
[tasks."mutation:helm:<chart>"]
run = '''
set -eu
helm-mutation-test <chart-dir> --since "$MUTATION_BASE" --report console,json --report-dir .tdd/mutation-helm-<chart>
jq -e '.tally.survived == 0 and .tally.noCoverage == 0' .tdd/mutation-helm-<chart>/mutation-report.json
'''
```

- **Les tests unitaires sont la cible de la mutation.**
  - En Go, gremlins lance les tests du paquet muté : les tests sous l'étiquette `container` et les tests d'acceptation d'un autre paquet ne comptent pas.
  - En Python, pytest ne lance que `tests/unit`, le répertoire unitaire déjà prévu par le gabarit (`tests/unit`, `tests/container`).
  - Un mutant qu'aucun test unitaire n'atteint fait échouer la tâche : `mutants_not_covered` en Go, `survived` en Python (essai : une fonction sans test compte en `survived`).
- **Quand rien n'a changé, la tâche sort en succès.**
  - En Go, `--diff` ne trouve aucun mutant : tous les compteurs sont à 0, et le `jq` passe.
  - En Python, la garde `test -n` s'en charge.
  - En Helm, helm-mutation-test sort en 0, et son rapport, sans mutant, passe le `jq`. C'est à confirmer par MUT-004.
- pytest-gremlins est une dépendance du groupe dev de `pyproject.toml` (`uv run`), pas un outil mise : c'est un plugin pytest, qui doit voir les dépendances du projet. Sa configuration va dans `[tool.pytest-gremlins]` (`paths`, `max_pardons`).
- En Python, la mutation se limite aux fichiers changés. Un fichier touché est muté en entier, anciens survivants compris. La CI garde le cache de pytest-gremlins entre deux runs.
- Les tâches `mutation:*` restent hors de `test` : `harness gate coverage --task test` ne les lance pas.
- La CI récupère l'historique complet (`fetch-depth: 0` sur GitHub, `GIT_DEPTH: 0` sur GitLab, déjà prévu dans `repo-tooling`) et passe `MUTATION_BASE=origin/<branche cible>`.
- Les tâches séquencent des commandes. `test -n "$targets" || exit 0` est une garde ordinaire (rien à muter, on passe). Le verdict est rendu par `jq`, un outil déclaré, et pas par du bash.

### Flux `stacked-tdd-mr` après le changement

Pour chaque tâche :
1. L'`acceptance-writer` écrit les tests de frontière depuis les critères (`test(<slug>): acceptance`).
2. L'`implementer` écrit le code **et ses tests unitaires** (`feat`).
3. L'orchestrateur lance `MUTATION_BASE=<branche N-1> mise run check` (dont `mutation`) et cite le code de sortie.
4. Quand la pile a un skill de pile, il lance aussi `harness gate coverage`.
5. Viennent ensuite la revue, la MR et la CI verte.

- Un mutant survivant ou non couvert est un constat pour l'implementer. Il ajoute le test unitaire qui le tue, ou il retire le code que rien ne demande (`UNTESTED`, comme aujourd'hui).
- Un mutant équivalent est exclu par la configuration de l'outil, et l'orchestrateur le consigne en `Ruling:`. Exemples : une grâce pytest-gremlins (plafonnée par `max_pardons`), `--exclude-mutators` / `--exclude` pour helm-mutation-test, `-E` pour gremlins.
- Disparaissent du skill :
  - l'étape 3 (porte red) ;
  - l'étape 4 (2) et l'étape 4b (mutant-writer, preuves, `--jobs`) ;
  - l'étape 5c (`characterize`) ;
  - le mode réécriture et `replay` ;
  - les lignes `CLAUSES`, `TESTS-CHANGE` et `TESTS.md`.
- Les critères d'acceptation dans les specs restent : ils disent à l'`acceptance-writer` quoi tester.
- `claude/CLAUDE.md` : « Unit tests are free scaffolding » devient « Unit tests are required: they are what `mise run mutation` measures ».

### Retrait dans le harnais

- `harness/cmd/harness/gate.go` : il ne reste que `gate coverage`. `harness gate red|green|characterize|replay` répond comme toute sous-commande inconnue.
- `harness/internal/app/gate/{gate,green,proofs,characterize,replay}.go` et `harness/internal/domain/tdd/` sont supprimés. `IsTestPath` part dans `harness/internal/domain/coverage`.
- Les tests d'acceptation du harnais liés à ces commandes partent (`harness/acceptance/clauses_test.go`, `tests_gardes_test.go`, ceux de `gate red/green`, `characterize` et `replay`). Les tests de `gate coverage` restent.
- `claude/agents/mutant-writer.md` est supprimé, et son déploiement `[dotfiles]` aussi. Les hooks et les tests bats qui le nomment (`agent-guard`, garde de modèle des sous-agents) sont mis à jour.
- `claude/TESTS.md` et `claude/skills/stacked-tdd-mr/templates/TESTS.md` sont supprimés. Les références sont réécrites dans `acceptance-writer.md`, `implementer.md`, `stack-py-app`, `hexagonal-go`, `README.md` et `_typos.toml`.
- Le harnais, qui est un module Go, reçoit lui-même `mutation:go`.

### Règle des scripts

Texte ajouté à `claude/CLAUDE.md` (section Development) et à `repo-tooling` :

> Un script reste en bash tant qu'il séquence des commandes : enchaînement, garde d'idempotence (« déjà fait, on passe »), contrôle de ses arguments. Dès qu'il décide d'un résultat par lui-même (il analyse une sortie ou un fichier, compare, calcule, rend un verdict), il passe en Python ou en Go, avec des tests lancés par `mise run test`.

Dans `claude/CLAUDE.md`, la phrase sur les tests d'acceptation (« proven red then green by `harness gate` and listed in `TESTS.md` ») devient : la preuve est la tâche `mutation` du dépôt, en CI, plus la porte de couverture.

### Mode complet, de temps en temps (ajout du 2026-11-27)

En temps normal, la mutation reste en mode diff. De temps en temps, on repasse sur tout le code, pour mesurer aussi ce que le diff ne regarde jamais.
- **`MUTATION_FULL=1`.** Les tâches `mutation:*` des trois fragments ignorent alors `MUTATION_BASE` et leur garde, puis mutent tout :
  - Go : `gremlins unleash` sans `--diff` ;
  - Python : les cibles sont tous les fichiers du paquet (`git ls-files`) ;
  - Helm : `helm-mutation-test` sans `--since`.

  Le verdict reste le même `jq` sur le rapport. Sans la variable, rien ne change.
- **Un workflow CI à part, `mutation-full`.** Il tourne à la demande seulement (`workflow_dispatch`), et lance `MUTATION_FULL=1 mise run mutation`. Il ne tourne pas sur les PR et ne bloque aucune fusion : un rouge est un signal, que l'on traite par des tests ajoutés ou des mutants équivalents exclus. `repo-tooling` le décrit pour GitHub et pour GitLab (job manuel, `when: manual`). Un dépôt peut y ajouter un `schedule` s'il en veut un.
- **Un rouge attendu au départ.** Le code écrit avant cette pile n'a jamais été mesuré.

### Choix de l'auteur (à relire)

- **Ordre des commits.** La convention reste `test(<slug>): acceptance` puis `feat(<slug>)`, mais rien ne vérifie plus cet ordre.
- **Code existant sans tests** (ancien `Mode: characterize`, et réécriture). L'`acceptance-writer` écrit les tests de frontière, et l'`implementer` les tests unitaires. Le verdict est la tâche `mutation`, lancée avec `MUTATION_BASE` sur le commit qui précède le code visé (en Python, les fichiers visés sont passés à `--gremlin-targets`).
- **Périmètre de la pile.** Le gabarit, le texte de `repo-tooling` et `stacked-tdd-mr` changent. Ce dépôt reçoit `mutation:go` pour `harness/`. Les autres dépôts adopteront les tâches à leur prochain passage par `repo-tooling`.
- **`MUTATION_BASE` dans une pile.** En CI, c'est la branche cible de la MR, donc la branche parente pour une MR empilée.
- **Plusieurs charts.** Une tâche `mutation:helm:<chart>` par chart, écrite à la main.
- **Tests d'acceptation MUT-002 à MUT-004.** Ils tournent dans le `test` de ce dépôt, sur des dépôts jetables construits avec les fragments. Les outils sont téléchargés par mise.
- **Inventaire du retrait.** Un `grep` sur tout le dépôt fixe la liste, au-delà des fichiers cités ici. Il cherche `harness gate red|green|characterize|replay`, `mutant-writer`, `TESTS.md`, `CLAUSES` et `TESTS-CHANGE`.
- **Règle des scripts.** Elle s'applique au code nouveau, y compris dans cette pile.

### Laissé au constructeur

- La version exacte de `jq` à épingler (`mise ls-remote jq`).
- Le nom exact des tests d'acceptation du harnais à supprimer, après inventaire.
- La forme du cache pytest-gremlins en CI (GitHub `actions/cache` avec `if: always()`, GitLab `cache:`).

## Suites (hors de cette pile)

Relecture des scripts bash du dépôt avec la règle (2026-11-27) :
- `.config/lint-harness` enfreint la règle : il compare `config.toml` aux skills présents et rend un verdict (`:9-11`), sans test.
- `tasks/bootstrap` est conforme : ses `if` sont des gardes d'idempotence (`:13`, `:80`, `:87`), et sa boucle séquence les complétions (`:125`).
- `claude/skills/docker-build-strategies/scripts/verify-build.sh` est conforme : ses `if` contrôlent ses arguments (`:13`, `:18`).
- `claude/hooks/tests/helpers.bash` est un utilitaire de tests bats, pas un script livré.

## Critères d'acceptation

Frontière : la ligne de commande `harness`, les tâches mise des fragments `repo-tooling` lancées sur des dépôts jetables, et le déploiement `test:dotfiles`.

### MUT-001 — Les commandes de preuve TDD n'existent plus
- **Étant donné** le binaire `harness` construit depuis la branche
- **Quand** on lance `harness gate red`, `harness gate green`, `harness gate characterize` puis `harness gate replay`
- **Alors**
  - chacune sort avec le code d'une sous-commande inconnue, et le dit sur la sortie d'erreur
  - `harness gate coverage` reste disponible

### MUT-002 — `mutation:go` échoue sur un mutant Go survivant ou non couvert par un test unitaire
- **Étant donné** un dépôt Go jetable avec le fragment `mutation-go.toml`, et une fonction ajoutée depuis `MUTATION_BASE`
- **Quand** on lance `mise run mutation:go`
- **Alors**
  - si le test unitaire du paquet appelle la fonction sans vérifier son résultat, la tâche sort en échec
  - si la fonction n'est appelée que par un test qui lance le binaire compilé, la tâche sort en échec
  - si le test unitaire vérifie le résultat, la tâche sort en succès
  - si aucun fichier Go n'a changé depuis `MUTATION_BASE`, la tâche sort en succès

### MUT-003 — `mutation:py` échoue sur un mutant Python survivant ou non couvert par un test unitaire
- **Étant donné** un dépôt Python jetable (uv, pytest, pytest-gremlins en dépendance dev) avec le fragment `mutation-py.toml`, et un fichier du paquet modifié depuis `MUTATION_BASE`
- **Quand** on lance `mise run mutation:py`
- **Alors**
  - si un test de `tests/unit` appelle une fonction de ce fichier sans vérifier son résultat, la tâche sort en échec
  - si aucune fonction de ce fichier n'est testée dans `tests/unit`, la tâche sort en échec
  - si les tests de `tests/unit` vérifient les résultats, la tâche sort en succès
  - si aucun fichier Python du paquet n'a changé depuis `MUTATION_BASE`, la tâche sort en succès

### MUT-004 — `mutation:helm:<chart>` échoue sur un mutant Helm survivant ou non rendu
- **Étant donné** un chart jetable avec une suite helm-unittest, le fragment `mutation-helm.toml`, et un template modifié depuis `MUTATION_BASE`
- **Quand** on lance `mise run mutation:helm:<chart>`
- **Alors**
  - si une suite rend ce template sans vérifier le champ modifié, la tâche sort en échec
  - si aucune suite ne rend le template modifié, la tâche sort en échec
  - si la suite vérifie chaque champ du template modifié (helm-mutation-test mute tout le fichier touché, pas seulement la ligne changée), la tâche sort en succès

### MUT-005 — Le harnais ne déploie plus l'agent `mutant-writer`
- **Étant donné** la configuration `[dotfiles]` de la branche
- **Quand** `test:dotfiles` déploie le harnais dans un HOME jetable
- **Alors**
  - `~/.claude/agents/mutant-writer.md` n'existe pas
  - `~/.claude/agents/acceptance-writer.md` et `implementer.md` existent

### MUT-007 — `MUTATION_FULL=1` mute tout le code, pas seulement le diff
- **Étant donné** pour chacun des trois fragments (Go, Python, Helm) un dépôt jetable dont le code non vérifié par les tests unitaires (pour Helm : un champ de template non vérifié par la suite) est déjà dans `MUTATION_BASE`, et rien n'a changé depuis
- **Quand** on lance la tâche `mutation:*` du fragment, sans puis avec `MUTATION_FULL=1`
- **Alors**
  - sans `MUTATION_FULL`, la tâche sort en succès
  - avec `MUTATION_FULL=1`, la tâche sort en échec
