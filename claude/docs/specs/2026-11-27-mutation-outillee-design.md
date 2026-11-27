# Mutation outillée : les dépôts prouvent leurs tests avec gremlins, pytest-gremlins et helm-mutation-test

Statut : proposé, 2026-11-27. ADR : `docs/adr/0003-la-preuve-des-tests-est-une-mutation-outillee-dans-chaque-depot.md`.

## Intention

Aujourd'hui, la valeur d'un test est prouvée par le harnais : `harness gate red` montre le test d'acceptation rouge, puis l'agent `mutant-writer` écrit un mutant par clause de critère et `harness gate green --proofs` vérifie que chaque mutant est tué (`claude/skills/stacked-tdd-mr/SKILL.md:11-14`). Cette chaîne est lente et dépend de l'IA à chaque maillon : un agent écrit les mutants, l'orchestrateur tranche `SURVIVED`, `REDUNDANT`, `NO-MUTANT`.

L'utilisateur veut réduire cette dépendance : la preuve devient le travail d'outils de mutation standard, déclarés dans chaque dépôt, lancés par `mise` en local comme en CI. L'IA reçoit une boîte à outils et s'en sert ; elle n'est plus elle-même la preuve. N'importe qui peut lancer `mise run mutation` sans Claude.

Deuxième demande, liée : un script ne reste en bash que s'il est une suite de commandes sans logique ; sinon il est écrit en Python ou en Go, avec des tests.

## Objectifs

- Chaque dépôt Go, Python ou Helm a une tâche `mutation` qui échoue quand un mutant survit au-dessus du seuil du dépôt :
  - Go : gremlins ;
  - Python : pytest-gremlins ;
  - Helm : helm-mutation-test (les charts ont déjà leurs suites helm-unittest, exigées par les skills Helm).
- `check`, donc la CI, lance `mutation`. La CI verte est le verdict de la merge request.
- Le harnais perd la preuve TDD : `harness gate red`, `green`, `characterize`, `replay`, le domaine `tdd`, l'agent `mutant-writer` et les fichiers `TESTS.md` disparaissent.
- `stacked-tdd-mr` devient plus court : critères, tests écrits depuis les critères, code, `mise run check`, couverture, revue, MR.
- La règle des scripts est écrite dans `claude/CLAUDE.md` et dans `repo-tooling`.

## Non-objectifs

- La porte de couverture `harness gate coverage` reste telle quelle (choix de l'utilisateur : « couverture gardée »). Elle est déjà déterministe et ne lit ni `TESTS.md` ni slug (`SKILL.md:15`) ; seule sa dépendance à `tdd.IsTestPath` (`harness/internal/app/gate/coverage.go:62`) est déplacée.
- Pas de mutation pour OpenTofu, Kyverno, Ansible ni les autres livrables du catalogue : aucun outil n'a été retenu ; ils gardent leurs tests natifs.
- Pas de migration des scripts bash existants qui enfreignent la nouvelle règle dans cette pile : ils sont inventoriés (section « Suites »).
- Pas de nouveau binaire ni de nouvelle commande `harness` pour piloter les outils (option 2 écartée).

## Anti-objectifs

- Un dépôt dont la tâche `mutation` ne tourne qu'avec le poste de l'utilisateur (outil installé globalement, plugin Helm installé à la main, `harness` requis) : échec, même si la CI passe ailleurs.
- Un seuil que l'orchestrateur abaisse pour passer une tâche : le seuil se change dans une merge request à part, sur décision de l'utilisateur.
- Réintroduire un agent qui juge les survivants à la place de l'outil.

## Contraintes (faits vérifiés le 2026-11-27)

- Aucun des trois outils n'est au registre mise ni à aqua (`mise registry`, 1077 entrées ; `aquaproj/aqua-registry/pkgs/...` en 404).
- **gremlins** v0.6.0 (2025-12-06, Apache-2.0) : `gremlins unleash --diff <ref>` ne mute que le code changé ; `--threshold-efficacy N` sort en code 10 si `KILLED/(KILLED+LIVED)` est sous N ; `--threshold-mcover N` en code 11 ; `-o file.json`. Pas de statut « équivalent ». Archive `github:` avec binaire `gremlins` à la racine (vérifié par extraction). Source : https://github.com/go-gremlins/gremlins/blob/main/docs/docs/usage/commands/unleash/index.md
- **pytest-gremlins** 1.15.0 (PyPI 2026-11-26, MIT, projet créé en 12/2025) : plugin pytest (`pytest --gremlins`), parallèle, cache par empreinte de contenu (`--gremlin-cache`). Pas de seuil : la doc le dit (`docs/cookbook/ci-integration.md:12`) et propose `jq` sur le rapport. Pas de mode diff : `--gremlin-targets` reçoit une liste de fichiers ; le code changé se limite donc au fichier. Mutants équivalents « graciés » dans le code, plafonnés par `--gremlin-max-pardons-pct`. Essai jetable du 2026-11-27 (paquet de 2 fonctions, 1 test sans assertion) : 4 mutants, 2 survivants, **code de sortie 0** ; rapport `coverage/gremlins/gremlins.json`, `summary = {total, zapped, survived, timeout, error, pardoned, percentage}` (`percentage` = 50.0) ; `--gremlin-targets=` vide mute tout le paquet (4 mutants). Source : https://github.com/mikelane/pytest-gremlins
- **Écartés pour Python** : mutmut 3.8.1 (ni seuil ni diff, mute tout le paquet) ; cosmic-ray 8.9.0 (seuil `cr-rate --fail-over` et filtre `cr-filter-git` natifs, mais relance la suite à chaque mutant et ignore `--fail-over 0`). Choix de l'utilisateur : pytest-gremlins.
- **helm-mutation-test** v1.2.0 (2026-10-22, un seul mainteneur, Apache-2.0) : embarque helm-unittest ; `--since <ref>` ; `--threshold N` (code 1 sous le seuil, 2 si les suites échouent déjà) ; score `killed/(killed+survived)`, équivalents détectés et exclus ; un changement de `values.yaml`, d'un partiel `_*.tpl` ou de `Chart.yaml` déclenche un run complet. Le binaire de release fonctionne seul, hors Helm (`--help` lancé). Source : https://github.com/mmpyro/helm-mutation-test/blob/main/docs/cli-reference.md
- Règle `repo-tooling` : tout outil est déclaré dans le `mise.toml` du dépôt, version exacte ; les tâches sont inline.

## Approches considérées

1. **Outils dans chaque dépôt, la porte disparaît** (retenue par l'utilisateur). Tâche `mutation` par pile, verdict en CI ; `gate red/green/...`, `mutant-writer`, `TESTS.md` supprimés. Compromis : plus rien ne vérifie que le test a été écrit avant le code.
2. **Le harnais enveloppe les outils.** `harness gate` lance les outils et lit leurs rapports. Compromis : on garde du code maison et un dépôt dépend encore du harnais pour sa preuve.
3. **Option 1 plus une porte red minimale.** Compromis : garde la discipline « test d'abord », au prix d'un reste de harnais et de l'agent qui l'orchestre.

## Conception

### Boîte à outils dans `repo-tooling`

Le gabarit `claude/skills/repo-tooling/templates/mise.toml` et le texte du skill gagnent, selon la pile détectée :

```toml
[tools]
"github:go-gremlins/gremlins" = "0.6.0"          # Go
"github:mmpyro/helm-mutation-test" = "1.2.0"     # Helm (binaire seul, pas de plugin Helm)
jq = "<version exacte>"                          # Python : lecture du score pytest-gremlins

[env]
MUTATION_BASE = "origin/main"   # la CI le remplace par la branche cible de la MR
MUTATION_THRESHOLD = "100"

[tasks.mutation]
description = "Run every mutation:* task"
depends = ["mutation:*"]

[tasks."mutation:go"]
run = 'gremlins unleash --diff "$MUTATION_BASE" --threshold-efficacy "$MUTATION_THRESHOLD" -o .tdd/mutation-go.json'

[tasks."mutation:py"]
run = '''
set -eu
targets=$(git diff --name-only --diff-filter=AMR "$MUTATION_BASE" -- '<paquet>/*.py' | paste -sd, -)
test -n "$targets" || exit 0   # sans fichier changé, --gremlin-targets vide muterait tout le paquet
uv run pytest -q --gremlins --gremlin-cache --gremlin-targets="$targets" --gremlin-report=json
jq -e --argjson t "$MUTATION_THRESHOLD" '.summary.percentage >= $t' coverage/gremlins/gremlins.json
'''

[tasks."mutation:helm"]
run = 'helm-mutation-test <chart> --since "$MUTATION_BASE" --threshold "$MUTATION_THRESHOLD" --report console,json --report-dir .tdd/mutation-helm'

[tasks.check]
depends = ["lint", "test", "mutation"]
```

- pytest-gremlins est une dépendance du groupe dev de `pyproject.toml` (`uv run`), pas un outil mise : c'est un plugin pytest qui doit voir les dépendances du projet. Sa configuration va dans `[tool.pytest-gremlins]` (`paths`, `max_pardons`).
- Python se limite aux fichiers changés : un fichier touché est muté en entier, ses anciens survivants compris. La CI garde le cache de pytest-gremlins entre deux runs.
- pytest-gremlins sort en 0 même avec des survivants : seul le `jq` fait échouer la tâche.
- Les tâches `mutation:*` restent hors de `test` : `harness gate coverage --task test` ne les lance pas.
- La CI récupère l'historique complet (`fetch-depth: 0` sur GitHub, `GIT_DEPTH: 0` sur GitLab, déjà dans `repo-tooling`) et passe `MUTATION_BASE=origin/<branche cible>`.
- Les lignes `run` sont des commandes sans logique, sauf la garde `test -n "$targets" || exit 0` de `mutation:py`, qui compense un comportement vérifié de l'outil (choix de l'auteur, voir plus bas).

### Flux `stacked-tdd-mr` après le changement

Par tâche : `acceptance-writer` écrit les tests depuis les critères (commit `test(<slug>): acceptance`), puis `implementer` écrit le code (`feat`). Ensuite, l'orchestrateur lance `mise run check` (dont `mutation`) et cite le code de sortie. Il lance aussi `harness gate coverage` quand la pile a un skill de pile. Viennent ensuite la revue, la MR et la CI verte.

- Un mutant survivant est un constat pour l'implementer : il ajoute un test qui le tue, ou il retire le code que rien ne demande (`UNTESTED` comme aujourd'hui).
- Un mutant équivalent est exclu par une exclusion d'outil, jamais en baissant le seuil, et l'orchestrateur la consigne en `Ruling:`. Exemples d'exclusion : une grâce pytest-gremlins (plafonnée par `max_pardons`), `--exclude-mutators` / `--exclude` pour helm-mutation-test, `-E` pour gremlins.
- Disparaissent du skill : les étapes 3 (porte red), 4 (2) et 4b (mutant-writer, preuves, `--jobs`), 5c (`characterize`), le mode réécriture/`replay`, les lignes `CLAUSES`, `TESTS-CHANGE`, `TESTS.md`.
- Les critères d'acceptation dans les specs restent : ils disent à l'`acceptance-writer` quoi tester.

### Retrait dans le harnais

- `harness/cmd/harness/gate.go` : il ne reste que `gate coverage` ; `harness gate red|green|characterize|replay` répond comme toute sous-commande inconnue.
- `harness/internal/app/gate/{gate,green,proofs,characterize,replay}.go` et `harness/internal/domain/tdd/` sont supprimés. `IsTestPath` part dans `harness/internal/domain/coverage`.
- Les tests d'acceptation du harnais liés à ces commandes partent (`harness/acceptance/clauses_test.go`, `tests_gardes_test.go` et ceux de `gate red/green`). Les tests de `gate coverage` restent.
- `claude/agents/mutant-writer.md` est supprimé, et son déploiement `[dotfiles]` aussi. Les hooks et tests bats qui le nomment (`agent-guard`, garde de modèle des sous-agents) sont mis à jour.
- `claude/TESTS.md` et `claude/skills/stacked-tdd-mr/templates/TESTS.md` sont supprimés. Les références dans `acceptance-writer.md`, `implementer.md`, `stack-py-app`, `hexagonal-go`, `README.md` et `_typos.toml` sont réécrites.
- Le harnais, qui est un module Go, reçoit lui-même `mutation:go`.

### Règle des scripts

Texte ajouté à `claude/CLAUDE.md`, section Development, et à `repo-tooling` :

> Un script ou une tâche reste en bash tant qu'il enchaîne des commandes sans logique propre. Dès qu'il contient une condition, une boucle, une analyse de texte ou un calcul, il est écrit en Python ou en Go, avec des tests lancés par `mise run test`. `set -eu`, `mkdir -p` et une redirection ne sont pas de la logique ; un `if`, un `for`, un `case`, `grep | sed` sur une sortie à interpréter en sont.

La phrase de `claude/CLAUDE.md` sur les tests d'acceptation (« proven red then green by `harness gate` and listed in `TESTS.md` ») devient : la preuve est la tâche `mutation` du dépôt, en CI.

### Choix de l'auteur (à relire)

- **Ordre des commits.** `test(<slug>): acceptance` puis `feat(<slug>)` restent la convention. L'`acceptance-writer` écrit les tests sans lire le code, l'`implementer` écrit le code. Rien ne vérifie plus cet ordre.
- **Code existant sans tests** (ancien `Mode: characterize` et réécriture). L'`acceptance-writer` écrit des tests sur le code existant. Le verdict est la tâche `mutation` lancée avec `MUTATION_BASE` sur le commit qui précède le code visé ; pour Python, les fichiers visés sont passés à `--gremlin-targets`. Aucune commande du harnais n'est nécessaire.
- **Périmètre de la pile.** Le gabarit, le texte de `repo-tooling` et `stacked-tdd-mr` changent. Ce dépôt reçoit `mutation:go` pour `harness/`. Les autres dépôts adoptent les tâches à leur prochain passage par `repo-tooling`.
- **`MUTATION_BASE` dans une pile.** En CI, c'est la branche cible de la MR, donc la branche parente pour une MR empilée. En local, `stacked-tdd-mr` lance `MUTATION_BASE=<branche N-1> mise run mutation`.
- **Garde de `mutation:py`.** Une seule condition, `test -n "$targets" || exit 0`, reste en bash : sans elle, un diff sans fichier Python mute tout le paquet (essai du 2026-11-27). Un script Python pour une ligne serait plus lourd que la règle ne le vise.
- **Plusieurs charts.** Il faut une tâche `mutation:helm:<chart>` par chart, écrite à la main. Une boucle serait de la logique.
- **Tests d'acceptation MUT-002 à MUT-004.** Ils tournent dans `test` de ce dépôt. Les outils sont déclarés dans son `mise.toml` et téléchargés par mise, comme les autres outils en CI (runner Linux).
- **Inventaire du retrait.** Un `grep` sur tout le dépôt (`harness gate red|green|characterize|replay`, `mutant-writer`, `TESTS.md`, `CLAUSES`, `TESTS-CHANGE`) fixe la liste, au-delà des fichiers cités ici. La section « Direct fix » de `claude/CLAUDE.md` et de `stacked-tdd-mr` perd sa mention de `TESTS.md`.
- **Règle des scripts.** Elle s'applique au code nouveau, y compris dans cette pile.
- **Licence de helm-mutation-test.** Apache-2.0, lue le 2026-11-27 dans le fichier `LICENSE` du dépôt.

### Laissé au constructeur

- La version exacte de `jq` à épingler (`mise ls-remote jq`).
- Le nom exact des tests d'acceptation du harnais à supprimer, après inventaire.
- La forme du cache pytest-gremlins en CI (GitHub `actions/cache` avec `if: always()`, GitLab `cache:`).

## Suites (hors de cette pile)

Scripts bash qui ont de la logique et enfreignent la nouvelle règle :
- `tasks/bootstrap` (152 lignes, conditions) ;
- `.config/lint-harness` (24 lignes) ;
- `claude/skills/docker-build-strategies/scripts/verify-build.sh` (34 lignes).

`claude/hooks/tests/helpers.bash` est un utilitaire de tests bats, à juger au même moment.

## Critères d'acceptation

Frontière : la ligne de commande `harness`, les tâches mise du gabarit `repo-tooling` lancées sur des dépôts jetables, et le déploiement `test:dotfiles`.

### MUT-001 — Les commandes de preuve TDD n'existent plus
- **Étant donné** le binaire `harness` construit depuis la branche
- **Quand** on lance `harness gate red`, `harness gate green`, `harness gate characterize` puis `harness gate replay`
- **Alors**
  - chacune sort avec le code d'une sous-commande inconnue et le dit sur la sortie d'erreur
  - `harness gate coverage` reste disponible

### MUT-002 — Un mutant Go survivant dans le code changé fait échouer `mutation:go`
- **Étant donné** un dépôt Go jetable avec les tâches du gabarit, une fonction ajoutée depuis `MUTATION_BASE` et un test qui l'appelle sans vérifier son résultat
- **Quand** on lance `mise run mutation:go`
- **Alors**
  - la tâche sort en échec
  - avec un test qui vérifie le résultat, la même tâche sort en succès

### MUT-003 — Un mutant Python survivant fait échouer `mutation:py`
- **Étant donné** un dépôt Python jetable (uv, pytest, pytest-gremlins en dépendance dev) avec les tâches du gabarit, un fichier Python modifié depuis `MUTATION_BASE` et une fonction de ce fichier dont le test ne vérifie pas le résultat
- **Quand** on lance `mise run mutation:py`
- **Alors**
  - la tâche sort en échec
  - avec un test qui vérifie le résultat, la même tâche sort en succès

### MUT-004 — Un mutant Helm survivant fait échouer `mutation:helm`
- **Étant donné** un chart jetable avec une suite helm-unittest qui rend un template sans vérifier un champ modifié depuis `MUTATION_BASE`
- **Quand** on lance `mise run mutation:helm`
- **Alors**
  - la tâche sort en échec
  - avec une assertion sur ce champ, la même tâche sort en succès

### MUT-005 — Le harnais ne déploie plus l'agent `mutant-writer`
- **Étant donné** la configuration `[dotfiles]` de la branche
- **Quand** `test:dotfiles` déploie le harnais dans un HOME jetable
- **Alors**
  - `~/.claude/agents/mutant-writer.md` n'existe pas
  - `~/.claude/agents/acceptance-writer.md` et `implementer.md` existent
