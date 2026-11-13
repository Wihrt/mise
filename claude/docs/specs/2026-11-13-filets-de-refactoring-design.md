# Filets de refactoring — spec

Date : 2026-11-13. Périmètre : le binaire `harness` (`harness/`, porte `gate`), `claude/TESTS.md`, `claude/agents/{implementer,acceptance-writer}.md`, `claude/skills/{stacked-tdd-mr,repo-tooling,hexagonal-go,hexagonal-py}/`, deux skills nouveaux (`compose-dev`, `twelve-factor`), `claude/docs/regles-generiques.md`, `config.toml` (`[dotfiles]`).

Remplace, dans `2026-11-10-harnais-deterministe-design.md`, la preuve « Code minimal » et le temps « refactor par retrait mesuré » (PRV-009, PRV-013, PRV-014, PRV-018).

## Objectif

Demande du 13/11 : « tester suffisamment l'application et en même temps ne pas craindre un refactoring, parce que les contraintes ont changé ou autre raison ».

Constat du 13/11, à l'usage : le harnais est très lent, et la cause est une seule preuve, le code minimal. Elle retire chaque bloc ajouté et relance la suite une fois par bloc, après que l'implementer a fait le même retrait à la main. L'idée était de n'avoir aucun code superflu ; elle est bonne, mais trop lente, et ce n'était pas le but. Le but est de pouvoir refactorer sans crainte.

Un refactoring est sûr quand trois filets tiennent :

| Filet | Ce qu'il garantit | État |
|---|---|---|
| Tests d'acceptation (`red`, `green`) | Le comportement promis existe | En place, inchangé |
| Mutants (un par clause du « Alors ») | Un test échoue quand ce comportement casse | En place, inchangé |
| Tests d'architecture | Les couches hexagonales ne se mélangent pas | À ajouter |

S'y ajoutent, pour une application Go ou Python (les deux langages de l'utilisateur, déployés sur Kubernetes) : les douze facteurs, SOLID, un `compose.yaml` avec watch pour retester à la volée, Bruno pour une API, Playwright pour un front.

## Décisions prises (13/11)

Décisions de l'utilisateur :

- D1 : la preuve du code minimal par retrait est jetée partout (porte, implementer, skill). Raison : trop lente.
- D2 : la consigne d'écriture reste : le plus petit code qui fait passer les tests, rien pour un cas hypothétique.
- D3 : des tests d'architecture garantissent l'architecture hexagonale.
- D4 : Go et Python applicatifs respectent hexagonal, SOLID et les douze facteurs ; `hexagonal-go` est corrigé (SOLID y manque).
- D5 : un skill dédié à Docker Compose, à côté de ceux sur Docker. Le compose avec watch est un prérequis pour l'applicatif Go et Python, pas pour l'infrastructure as code (Helm, OpenTofu, Ansible), qui a d'autres moyens de test.
- D6 : une API se teste avec Bruno, un front avec Playwright (déjà au catalogue depuis le 12/11).
- D7 : `repo-tooling` porte les contraintes nouvelles.
- D8 : façon de forcer, proposée par Claude et acceptée : architecture forcée par un contrôle rapide prouvé par un cas faux ; douze facteurs forcés par un socle de critères d'acceptation testés sur le conteneur lancé par le compose ; SOLID en règle écrite et à la revue ; la porte refuse quand le socle manque.

Choix de Claude (non discutés, à relire) :

- C1 : le contrôle d'architecture est un **test** qui porte un ID de critère, pas une tâche de lint à part. Annoncé à l'utilisateur comme « une tâche `lint:arch` » ; changé parce qu'un critère est prouvé par mutant, rejoué par `replay`, et que sa tâche est déjà tenue par `NOT WIRED`. Il tourne dans `test:go` ou `test:py`, en quelques secondes.
- C2 : la porte reconnaît un dépôt qui porte du code de production, pas un dépôt qui n'a qu'un manifeste pour ses tests (un chart ou un dépôt OpenTofu a un `pyproject.toml` pour pytest, ou un `go.mod` pour Terratest). À la racine du commit jugé : `go.mod` avec un dossier `cmd/` ou `internal/`, ou `pyproject.toml` avec un dossier `src/` → socle d'architecture ; avec en plus un `Dockerfile` → socle des douze facteurs. Limites connues : un dépôt dont le code est dans un sous-dossier n'est pas reconnu (cas du dépôt du harnais lui-même, `harness/go.mod`), ni une bibliothèque Go dont les paquets sont à la racine.
- C3 : les IDs du socle sont réservés et fixes : `ARC-001` à `ARC-003`, `FAC-001` à `FAC-005`.
- C4 : les options `--no-minimal`, `--unneeded-ok` et `--since` sont supprimées, pas tolérées : la porte les refuse comme toute option inconnue.
- C5 : le temps « refactor » reste, facultatif, et change de sens : améliorer la structure sous des tests verts, sans retrait mesuré.
- C6 : le watch n'est tenu par aucun contrôle de la porte (un critère « le fichier contient `develop.watch` » serait une recopie de configuration). Il est dans le gabarit et vérifié à la main au commit `chore(tooling)`.
- C7 : rien ne détecte « cette API n'a pas de test Bruno » : la porte ne sait pas qu'un critère parle de HTTP. La règle reste au catalogue.
- C9 : la liste des bibliothèques permises au domaine (ARC-001) est écrite dans le fichier du test d'architecture, donc protégée : l'implementer ne peut pas l'élargir, et une dépendance nouvelle du domaine est un changement de critère (`--ac-changed ARC-001`).
- C10 : aucun dépôt jetable n'est poussé sur une forge pour vérifier le job CI qui fournit Docker. La ligne GitHub est vérifiée sur le premier dépôt d'application réel ; la ligne GitLab est notée « unverified », comme l'image mise l'est déjà dans `repo-tooling`.
- C8 : noms des skills (`compose-dev`, `twelve-factor`), des tâches (`dev`, `lint:compose`, `test:container`, `test:api`, `test:ui`) et du gabarit (`templates/socle-application.md`).

## Ce qui n'est pas voulu

Hors périmètre :

- Mettre les dépôts existants en conformité. Le socle s'impose à chaque dépôt à sa prochaine pile, par la porte.
- Un contrôle de la porte pour SOLID, pour le watch, pour la présence de Bruno ou Playwright.
- Appliquer le socle au dépôt du harnais (non reconnu, voir C2).
- Toucher aux preuves « spécifique » (changement neutre) et « nécessaire » (redondance), à `replay`, à `--jobs`.

Échec même si ça marche :

- Un contrôle qui relance la suite de tests une fois par bloc, par fichier ou par ligne de code ajouté.
- Une durée de porte qui grandit avec la quantité de code écrit plutôt qu'avec le nombre de clauses.
- Une règle qui ne s'énonce qu'en nommant un projet (voir `docs/regles-generiques.md`).

## Conception

Trois chantiers, dans cet ordre. Le premier est indépendant et rend la vitesse tout de suite.

### 1. Retirer la preuve du code minimal

**Porte.** `green --proofs` ne retire plus aucun bloc. Disparaissent : `runMinimal` (`internal/app/gate/proofs.go`), `tdd.AddedBlocks` et son test unitaire, les champs `NoMinimal`, `UnneededOK`, `Since`, les options `--no-minimal`, `--unneeded-ok`, `--since`, les lignes `UNNEEDED`, `UNNEEDED ACCEPTED`, `MINIMAL SKIPPED`, et leur mention dans l'usage. `--since` ne servait qu'à cette preuve.

**Critères retirés.** PRV-009, PRV-013, PRV-014, PRV-018 : leurs entrées de `claude/TESTS.md` et leurs tests de `harness/acceptance/preuves_green_test.go` (`TestPRV009_*`, `TestPRV013_*`, `TestPRV014_*`, `TestPRV018_*`), l'aide `unneededOK` de `helpers_test.go`. PRV-017 (`--no-neutral`) reste ; le titre de sa section ne parle plus du code minimal. Le retrait passe par les règles du harnais lui-même (`Criterion change:` consigné, `--replaces` ou `--ac-changed`, `replay` avant la revue finale) ; la suite exacte des commandes est à établir dans le plan. Le nouveau critère PRV-020 donne le rouge de départ.

**Implementer** (`agents/implementer.md`). Restent : « the smallest production code that makes them pass », pas de code pour un cas hypothétique, les lignes `UNTESTED <élément> : <pourquoi>`, et la preuve par retrait d'un test ajouté après coup et vert d'emblée (un seul lancement). Disparaissent : le retrait bloc par bloc, `NOTHING TO REDUCE`, la consigne de dispatch `no measured removal`. Le refactor devient facultatif (C5) : un commit `refactor(<slug>)`, tests intouchés, quand la structure le demande (couche, interface, duplication). Réponse : `DONE <sha feat>` ou `DONE <sha feat> <sha refactor>`, puis les lignes `UNTESTED`.

**Skill** (`skills/stacked-tdd-mr/SKILL.md`). Étape 4 : vert, refactor facultatif, mutants sur le dernier commit. Étape 4b : par tâche, la preuve sensible seule, `--no-neutral` ; plus de `--no-minimal`. Étape 8 : plus de « Minimal code, once for the stack » ; restent `replay` et la revue finale. « Work without a test » et « Rebase and conflicts » : plus de `--since`.

**Autres textes.** `docs/regles-generiques.md` : les lignes dont le contrôle est `UNNEEDED` passent à « retiré le 13/11 : trop lent » et renvoient ici. `2026-11-10-harnais-deterministe-design.md` : une ligne en tête du paragraphe « Code minimal » et une sous « Rouge, vert, refactor » disent « retiré le 13/11, voir `2026-11-13-filets-de-refactoring-design.md` » ; le reste de cette spec n'est pas réécrit.

### 2. Skills, gabarits et socle

Aucun critère d'acceptation pour ces textes (règle de la spec du 10/11 : un test qui vérifie qu'un fichier contient une phrase est une recopie). Chaque affirmation technique d'un skill est vérifiée sur une application d'essai et notée « verified » avec la version de l'outil, comme dans les skills actuels. Deux applications d'essai, jetables et hors dépôt : une API Go et une API Python, chacune avec ses couches, son `Dockerfile`, son `compose.yaml`.

**Socle** (`skills/stacked-tdd-mr/templates/socle-application.md`, nouveau). Les critères que toute application porte, à recopier dans la spec du dépôt et à transcrire par l'acceptance-writer. Deux sections, deux slugs.

`architecture` — tout dépôt Go ou Python :

- **ARC-001** — Le domaine ne dépend que de lui-même. Étant donné le code du dépôt, quand le test d'architecture est lancé, alors aucun paquet du domaine n'importe, directement ou par un intermédiaire, un paquet de `app`, un adapter, un point d'entrée, ni une bibliothèque hors de la liste permise (bibliothèque standard, plus celles que le fichier de test énumère, voir C9).
- **ARC-002** — Les cas d'usage ne connaissent aucune technologie. Même forme : aucun paquet de `app` n'importe un adapter ni un point d'entrée.
- **ARC-003** — Seule la racine de composition assemble. Même forme : un adapter n'est importé que par la racine de composition (`cmd/<app>` en Go, `composition.py` en Python).

`douze-facteurs` — dépôt Go ou Python avec `Dockerfile` ; le produit est l'image, lancée par le `compose.yaml` du dépôt :

- **FAC-001** — Le port d'écoute vient de l'environnement (facteurs III et VII). Lancée avec un port autre que celui par défaut, l'application répond sur ce port.
- **FAC-002** — Une variable obligatoire absente arrête le démarrage (III). Le conteneur sort avec un code non nul ; sa sortie nomme la variable.
- **FAC-003** — Les logs sortent sur la sortie standard (XI). Après une requête, la sortie du conteneur porte la ligne de log de cette requête.
- **FAC-004** — L'arrêt est propre (IX). Sur SIGTERM, le conteneur sort avec le code 0 avant la fin du délai d'arrêt.
- **FAC-005** — Aucun état local (VI). Le service tourne avec un système de fichiers racine en lecture seule et un utilisateur non root, et sert son cas nominal.

Un critère qui ne s'applique pas (FAC-001 pour un worker sans port) est écarté par `--socle-exempt` après un `Ruling:`. Les facteurs I, II, IV, V, VIII, X, XII ne se testent pas ainsi : ils sont des règles écrites dans `twelve-factor` (II est déjà tenu par `repo-tooling` : toute dépendance déclarée dans le dépôt).

**Test d'architecture.** Exigences : liste blanche par couche ; fermeture transitive des imports (un intermédiaire ne masque rien) ; insensible aux alias d'import ; un résultat par critère, nommé avec l'ID, dans le rapport JUnit de `test:go` ou `test:py`. Pistes, à vérifier sur les applications d'essai et laissées au constructeur : en Go, un test qui lit `go list -deps` ; en Python, un test pytest sur le graphe d'imports (`grimp`, la bibliothèque d'`import-linter`). Les lints actuels (forbidigo, `banned-api` de ruff) restent comme retour rapide au commit ; leurs limites connues (alias, imports transitifs) sont celles que le test couvre.

**`hexagonal-go`.** Ajout d'une section « SOLID, in Go terms » (pendant de celle de `hexagonal-py`) ; « Enforcing the layers » décrit le test d'architecture et renvoie au socle ; renvoi vers `twelve-factor` et `compose-dev`.

**`hexagonal-py`.** Même test d'architecture, mêmes renvois.

**`twelve-factor`** (nouveau). Les douze facteurs en termes Go, Python et Kubernetes : pour chacun, ce que le code doit faire et qui le tient (un critère `FAC-…` du socle, un autre skill, ou la règle écrite seule).

**`compose-dev`** (nouveau). `compose.yaml` à la racine : le service de l'application construit le `Dockerfile` de production (parité, facteur X), `read_only: true`, utilisateur non root, réglages par `environment` ; les services d'appui (base, cache) avec `healthcheck` et `depends_on: condition: service_healthy` ; section `develop.watch` : reconstruction sur changement de source en Go, synchronisation et redémarrage en Python si elle est compatible avec `read_only` (à vérifier ; sinon reconstruction aussi), reconstruction sur changement du fichier de dépendances. Les tests lancent la pile sous un nom de projet et un port propres à l'exécution, pour que `--jobs` reste utilisable. Si le skill public `docker-compose-patterns` (cité par `docker-build-strategies`) existe, il est repris tel quel à côté et `compose-dev` ne fait que le compléter, comme `dockerfile-restricted` complète `docker-build-strategies`.

**`repo-tooling`.** Nouvelle section « Application Go ou Python » (dépôt avec `Dockerfile` ; l'infrastructure as code n'est pas concernée) : `compose.yaml` avec watch ; tâches `dev` (`docker compose up --watch`, hors `check`), `lint:compose` (`docker compose config --quiet`), `test:container` (socle des douze facteurs), `test:api` (Bruno) et `test:ui` (Playwright) quand le dépôt a une API ou un front ; chaque tâche de test a son rapport sous `.tdd/` ; gabarits `templates/compose.go.yaml` et `templates/compose.py.yaml`. Docker est requis sur le poste et en CI (C10).

**`acceptance-writer.md`.** Lignes « HTTP API » et « Web front » : le produit est démarré par le compose du dépôt (`docker compose up -d --wait`, puis `down -v`), par l'aide commune des tests. Deux lignes nouvelles : « Architecture (Go, Python) » et « Conteneur en exécution (douze facteurs) ».

**`stacked-tdd-mr`.** Setup, nouveau point : un dépôt Go ou Python sans socle commence sa pile par les tâches du socle, après `chore(tooling)` : `Mode: characterize` quand le code est déjà conforme, `atdd` sinon (le test d'architecture est rouge, l'implementer déplace le code). Avant de l'installer, vérifier qu'aucune entrée de `TESTS.md` ne porte déjà un ID `ARC-…` ou `FAC-…` pour autre chose : la porte ne lit que l'ID ; en cas de collision, s'arrêter et le dire. Étape 8 : le brief de la revue finale nomme le skill hexagonal du langage ; une remarque SOLID cite le code visé et passe par « Work without a test » (commit `refactor`, `green` rejouée avec les preuves).

**`config.toml`.** Une ligne `[dotfiles]` par skill nouveau (`mise run lint:harness` refuse un skill sans la sienne).

### 3. La porte exige le socle

Règle générique : un dépôt d'application porte son socle de critères ; une fonctionnalité ne s'ajoute pas avant lui.

Modes `red`, `green`, `characterize`. La porte lit la racine du commit jugé (C2), en déduit les IDs exigés (C3), et refuse par `SOCLE MISSING <ID>` chaque ID exigé absent de `TESTS.md` à ce commit. Deux exceptions : le slug jugé est lui-même un slug du socle (toutes ses entrées ont un ID du socle) ; l'ID est donné à `--socle-exempt <ID,...>` (après un `Ruling:` consigné : dépôt qui n'est pas une application, critère sans objet). La porte ne lit pas le texte des entrées du socle : leur valeur est prouvée par leurs mutants.

Le compose n'a pas de contrôle à lui : les entrées `FAC-…` nomment une tâche (`Commande`), `NOT WIRED` exige déjà qu'elle soit branchée, et cette tâche lance le compose.

### Coût

- Par tâche : un lancement de la tâche de test par mutant, comme aujourd'hui, et plus rien par bloc de code.
- Test d'architecture : quelques secondes dans `test:go` ou `test:py`.
- `test:container`, `test:api`, `test:ui` : un démarrage de pile par lancement. Ces tâches sont séparées de `test:go` et `test:py` pour que les mutants d'un slug de logique ne paient pas le conteneur. `mise run check` les lance toutes.
- Mise en place du socle sur un dépôt : deux tâches de pile, une fois.

## Critères d'acceptation

Préfixes : `PRV` (preuves), `SOC` (socle). Tests dans `harness/acceptance/`, tâche `mise run test`.

### PRV-020 — Un bloc qu'aucun test ne demande n'est plus jugé
- **Étant donné** un commit d'acceptation, un commit `feat` qui ajoute dans un fichier de production un bloc dont le retrait laisse tous les tests verts, et un dossier de preuves dont le mutant tue le test du critère
- **Quand** `harness gate green --proofs DIR --no-neutral` est lancé
- **Alors**
  - la sortie ne contient aucune ligne `UNNEEDED` ni `MINIMAL`, et se termine par `PASS`
  - relancé avec `--no-minimal`, la porte sort avec le code 2 (option inconnue)
- **Cas réel** : essai du 13/11 par l'utilisateur : porte « très lente », la preuve relance la suite une fois par bloc ajouté (PRV-018, pile MR 25 à 54)

### SOC-001 — Une fonctionnalité ne s'ajoute pas avant le socle d'architecture
- **Étant donné** un dépôt dont la racine porte `go.mod` et un dossier `internal/`, un `TESTS.md` sans entrée `ARC-001`, `ARC-002`, `ARC-003`, et un slug ordinaire avec son commit d'acceptation
- **Quand** `harness gate red` est lancé sur ce slug
- **Alors**
  - la sortie contient `SOCLE MISSING ARC-001`, `SOCLE MISSING ARC-002`, `SOCLE MISSING ARC-003` et se termine par `FAIL`
  - avec `pyproject.toml` et un dossier `src/` à la place, la sortie est la même
  - avec `pyproject.toml` et un dossier `tests/` sans dossier `src/`, la sortie ne contient aucune ligne `SOCLE MISSING`
  - avec `go.mod` sans dossier `cmd/` ni `internal/`, la sortie ne contient aucune ligne `SOCLE MISSING`
- **Cas réel** : demande du 13/11 : « ajouter des tests d'architecture pour garantir que l'architecture hexagonale est respectée » ; le catalogue de l'acceptance-writer met pytest (donc un `pyproject.toml`) dans les dépôts Helm, OpenTofu, Kyverno et Ansible, et Terratest (donc un `go.mod`) dans un dépôt OpenTofu

### SOC-002 — Un dépôt avec Dockerfile doit aussi le socle des douze facteurs
- **Étant donné** un dépôt dont la racine porte `go.mod`, un dossier `internal/` et `Dockerfile`, un `TESTS.md` qui a les trois entrées `ARC-…` et aucune entrée `FAC-…`, et un slug ordinaire
- **Quand** `harness gate red` est lancé sur ce slug
- **Alors**
  - la sortie contient `SOCLE MISSING FAC-001` à `SOCLE MISSING FAC-005`, aucune ligne `SOCLE MISSING ARC-…`, et se termine par `FAIL`
  - sans `Dockerfile` à la racine, la sortie ne contient aucune ligne `SOCLE MISSING`
- **Cas réel** : demande du 13/11 : douze facteurs à forcer, « je déploie quasiment que sur Kube »

### SOC-003 — Le socle s'installe et s'exempte
- **Étant donné** le dépôt de SOC-002
- **Quand** `harness gate red` est lancé
- **Alors**
  - sur un slug dont toutes les entrées sont des `FAC-…`, la sortie ne contient aucune ligne `SOCLE MISSING`
  - sur un slug ordinaire avec `--socle-exempt FAC-001`, la sortie contient `SOCLE MISSING FAC-002` et aucune ligne `SOCLE MISSING FAC-001`
- **Cas réel** : un worker sans port d'écoute n'a pas de FAC-001 ; l'installation du socle est elle-même une tâche jugée par la porte

### SOC-004 — Le socle est exigé à chaque mode
- **Étant donné** le dépôt de SOC-001, un slug ordinaire, son commit d'acceptation et un commit `feat` vert
- **Quand** `harness gate green` puis `harness gate characterize` sont lancés sur ce slug
- **Alors** chaque sortie contient `SOCLE MISSING ARC-001` et se termine par `FAIL`
- **Cas réel** : une tâche en `Mode: characterize` ne passe jamais par `red`

## Question ouverte (en attente de l'utilisateur)

La porte ne sait pas juger ce dépôt-ci. Elle lit `<repo>/TESTS.md` (`internal/app/gate/gate.go`, `prepare`) et `<sha>:mise.toml`, alors que ce dépôt a `claude/TESTS.md` et `.config/mise.toml` ; `--replaces` lit `<sha>~1:TESTS.md` à la racine git ; `test:go` écrit `.tdd/go.xml` et non `.tdd/junit.xml`. Le retrait des critères PRV-009, 013, 014, 018 « par les règles du harnais » et la preuve rouge puis verte de PRV-020 et SOC-001 à 004 ne peuvent donc pas passer par `harness gate` tel quel. À trancher : rendre d'abord la porte capable de juger ce dépôt (chantier préalable, avec ses critères), ou faire ces changements en TDD à la main (tests d'acceptation écrits et vus rouges avant le code, consignés), sans la porte.

## Laissé au constructeur

- L'outil du test d'architecture dans chaque langage, dans les exigences ci-dessus.
- L'outil des tests du socle des douze facteurs (tests natifs du langage qui pilotent `docker compose`).
- Le découpage en tâches et en piles, l'ordre des commits de retrait des critères PRV-009, 013, 014, 018.
- Le contenu exact des gabarits `compose`, une fois vérifié sur les applications d'essai.
- La forme des applications d'essai.
