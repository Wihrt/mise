# Filets de refactoring — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Do NOT use `stacked-tdd-mr` nor `harness gate` on this repository (spec, D9).

**Goal:** retirer du harnais la preuve du code minimal par retrait (lente), et ajouter les skills qui imposent, pour une application Go ou Python, tests d'architecture, douze facteurs, SOLID et compose avec watch.

**Architecture:** deux parties. A : le binaire `harness` perd `runMinimal` et ses trois options, avec les textes qui les citent (même PR, car `~/.claude` est fait de liens vers ce dépôt et le binaire est recompilé par `mise bootstrap --update`). B : des textes seulement (skills, gabarits, catalogue), dont chaque affirmation technique est vérifiée sur deux applications d'essai jetables avant d'être écrite.

**Tech Stack:** Go (binaire `harness/`, tests boîte noire `harness/acceptance/`, gotestsum), mise (`.config/mise.toml` : `test:go`, `lint:harness`, `check`), Docker Compose ≥ 2.22 (`develop.watch`), Bruno CLI, pytest-playwright, cocogitto.

**Spec:** `claude/docs/specs/2026-11-13-filets-de-refactoring-design.md`

## Global Constraints

- Aucun contrôle ne relance la suite de tests par bloc, par fichier ou par ligne de code ajouté.
- Options `--no-minimal`, `--unneeded-ok`, `--since` supprimées, pas tolérées : code de sortie 2.
- La consigne « the smallest production code that makes them pass » et les lignes `UNTESTED` restent dans `implementer.md`.
- TDD à la main sur ce dépôt : test d'acceptation commité et vu rouge (sortie citée) avant le code ; `mise run check` vert (dernière ligne citée) avant chaque commit de code ou de texte.
- Aucun critère d'acceptation ni test pour un texte. Une affirmation technique d'un skill n'est écrite qu'après avoir été vue sur une application d'essai, et porte « verified <date> » avec la version de l'outil.
- Skills et agents en anglais, dans le style des skills actuels (frontmatter `name`, `description` qui commence par « Use when », phrases courtes) ; specs, `TESTS.md`, gabarits de critères et `regles-generiques.md` en français.
- Une règle s'énonce sans nommer un projet (`claude/docs/regles-generiques.md`).
- IDs réservés du socle : `ARC-001` à `ARC-003`, `FAC-001` à `FAC-005`.
- Noms fixés : skills `repo-stack`, `stack-go-app`, `stack-py-app`, `compose-dev`, `twelve-factor` ; tâches `dev`, `lint:compose`, `test:container`, `test:api`, `test:ui` ; gabarit `templates/socle-application.md`.
- Commits conventionnels (`cog`), sans attribution d'IA ajoutée par un sous-agent. Ne jamais fusionner ; pousser et ouvrir une PR seulement à la demande de l'utilisateur.
- Applications d'essai et notes hors du dépôt, dans le dossier de travail temporaire de la session (`<essais>/go-api`, `<essais>/py-api`, `<essais>/NOTES.md`). Aucun dépôt jetable poussé sur une forge.

## Review Focus

1. Une ligne de commande ancienne (registre, procédure de rebase) qui porte encore `--no-minimal`, `--since` ou `--unneeded-ok` : on attend un refus qui nomme l'option, code 2, pas une acceptation muette. → test `TestPRV020_RetiredOptionsAreRefused` (tâche 1).
2. Binaire et textes déployés à des versions différentes : le skill passerait `--no-minimal` à un binaire qui le refuse. → tâches 1 et 2 dans la même PR ; étape de recherche des mentions restantes (tâche 2, étape 6).
3. Un import interdit caché derrière un alias ou un paquet intermédiaire : le test d'architecture doit échouer. → mutants M2 et M3 (tâche 3, étape 3).
4. Un dépôt d'infrastructure qui a un `pyproject.toml` pour pytest ou un `go.mod` pour Terratest : aucune stack ne doit lui être imposée. → cas D3 et D4 (tâche 6, étape 4).
5. Deux lancements simultanés de `test:container` (preuves avec `--jobs`) : pas de collision de nom de projet ni de port. → vérification V4 (tâche 4, étape 2).

---

## Partie A — retirer la preuve du code minimal (branche `feat/sans-code-minimal`, depuis `origin/main`)

### Task 1: La porte ne juge plus le code minimal (PRV-020)

**Files:**
- Modify: `harness/acceptance/preuves_green_test.go`, `harness/acceptance/helpers_test.go` (`greenRun`, `runGreen`)
- Modify: `claude/TESTS.md`
- Modify: `harness/internal/app/gate/proofs.go` (`runMinimal`), `harness/internal/app/gate/green.go` (branche `NoMinimal`), `harness/internal/app/gate/gate.go` (`Options`), `harness/cmd/harness/gate.go` (`gateUsage`, `parseGate`)
- Modify: `harness/internal/domain/tdd/proofs.go` (`Block`, `AddedBlocks`, `hunkNew`), `harness/internal/domain/tdd/tdd_test.go` (`TestAddedBlocks`)

**Interfaces:**
- Consumes: aides de test existantes `feature(r, content) (test, feat string)`, `r.runGreen(test, feat, greenRun) (int, string)`, `with`, `without`, `ends`.
- Produces: `harness gate green` sans les options `--no-minimal`, `--unneeded-ok`, `--since` ; plus aucune ligne `UNNEEDED`, `UNNEEDED ACCEPTED`, `MINIMAL SKIPPED`. La tâche 2 écrit les textes d'après ce contrat.

- [ ] **Step 1: Écrire les tests rouges** à la fin de `preuves_green_test.go`

```go
// Line 4 ("extra") is a block no test needs: it is no longer judged.
func TestPRV020_UnneededBlockIsNoLongerJudged(t *testing.T) {
	r := newRepo(t)
	test, feat := feature(r, "header\nneeded\nmiddle\nextra\n")
	code, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, neutral: neutral{none: true}, options: []string{"--no-neutral"}})
	with(t, out, "KILLED CALC-003")
	without(t, out, "UNNEEDED", "MINIMAL")
	ends(t, code, out, 0, "PASS")
}

func TestPRV020_RetiredOptionsAreRefused(t *testing.T) {
	for _, option := range [][]string{{"--no-minimal"}, {"--since", "HEAD"}, {"--unneeded-ok", prod + ":4-4"}} {
		r := newRepo(t)
		test, feat := feature(r, "header\nneeded\nmiddle\nextra\n")
		code, out := r.runGreen(test, feat, greenRun{noExempt: true, proofs: true, neutral: neutral{none: true}, options: append([]string{"--no-neutral"}, option...)})
		if code != 2 {
			t.Errorf("%s: exit code = %d, want 2\n%s", option[0], code, out)
		}
		with(t, out, strings.TrimLeft(option[0], "-"))
	}
}
```

Si `r.gate` ne capture pas la sortie d'erreur, retirer la ligne `with(...)` du second test et n'asserter que le code.

- [ ] **Step 2: Ajouter l'entrée de `claude/TESTS.md`**, à la fin du fichier

```markdown
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
```

- [ ] **Step 3: Voir les tests rouges**

Run: `go test -C harness ./acceptance -run 'TestPRV020' -count=1`
Expected: FAIL. Le premier test montre `UNNEEDED src/calc.txt:4-4` et `FAIL` ; le second un code de sortie autre que 2 pour les trois options. Citer la sortie.

- [ ] **Step 4: Commit** `test(sans-code-minimal): acceptance` (les deux fichiers ci-dessus)

- [ ] **Step 5: Retirer la preuve du binaire**

- `proofs.go` : supprimer `runMinimal`.
- `green.go` : dans le cas `r.o.Proofs != ""`, supprimer le bloc `if r.o.NoMinimal { … } else { … }` ; `NEUTRAL SKIPPED` reste.
- `gate.go` : supprimer les champs `NoMinimal`, `UnneededOK`, `Since` de `Options`.
- `cmd/harness/gate.go` : supprimer les trois déclarations d'option ; la ligne d'usage devient `[--proofs DIR [--jobs N] [--no-neutral]]`. `--spec` reste accepté par `green`.
- `domain/tdd/proofs.go` : supprimer `Block`, `AddedBlocks`, `hunkNew` et les imports devenus inutiles ; `tdd_test.go` : supprimer `TestAddedBlocks`.

- [ ] **Step 6: Retirer les critères PRV-009, PRV-013, PRV-014, PRV-018**

- `preuves_green_test.go` : supprimer `twoBlocks`, `TestPRV009_UnneededBlockIsRefused`, `TestPRV009_UnneededOKAcceptsTheBlock`, `TestPRV009_DeletionAndTestFileAreNotJudged`, `TestPRV013_UnneededBlockAddedBetweenTwoAcceptanceCommits`, `TestPRV014_SpecLinesAddedBetweenTwoAcceptanceCommitsAreNotUnneeded`, `TestPRV018_MinimalCodeIsSkippedOnRequest` ; mettre à jour le commentaire d'en-tête (slugs restants : preuves-nouveau, revue-finale PRV-012, preuves-optionnelles PRV-017, sans-code-minimal PRV-020). `prod`, `neededTask`, `prodBase`, `feature` restent.
- `helpers_test.go` : supprimer le champ `unneededOK` de `greenRun` et sa boucle dans `runGreen`.
- `claude/TESTS.md` : supprimer la section `## code-minimal` entière (PRV-009), l'entrée `### PRV-013` de la section `revue-finale`, la section `## spec-hors-code` entière (PRV-014), l'entrée `### PRV-018` ; le titre de section devient `## preuves-optionnelles — Le changement neutre se saute sur demande`.

- [ ] **Step 7: Voir tout vert**

Run: `go test -C harness ./acceptance -run 'TestPRV020|TestPRV017|TestPRV007|TestPRV012' -count=1` puis `mise run check`
Expected: PASS ; `check` vert. Citer la dernière ligne de chacun.

- [ ] **Step 8: Commit** `feat(sans-code-minimal)!: la porte ne juge plus le code minimal`

### Task 2: Textes qui citaient la preuve par retrait

**Files:**
- Modify: `claude/agents/implementer.md` (lignes « Refactor by measured removal », « Dispatch says `no measured removal` », « Reply »)
- Modify: `claude/skills/stacked-tdd-mr/SKILL.md` (étapes 4, 4b, 8 ; « Work without a test » ; « Rebase and conflicts »)
- Modify: `claude/docs/regles-generiques.md`, `claude/docs/specs/2026-11-10-harnais-deterministe-design.md`

**Interfaces:**
- Consumes: le contrat de la tâche 1.
- Produces: réponse de l'implementer `DONE <feat sha>` ou `DONE <feat sha> <refactor sha>`, suivie des lignes `UNTESTED <element> : <why>`.

- [ ] **Step 1: `implementer.md`**

- Garder tels quels : « Write the smallest production code that makes them pass. No scope beyond the tests. », la règle sur les cas hypothétiques, la ligne « A test added afterwards that is green at once is proved by removal ».
- Remplacer la ligne « Refactor by measured removal, in three steps… » par : refactor facultatif (après `feat(<slug>)`, quand la structure le demande : couche franchie, interface à scinder, duplication ; commit `refactor(<slug>): <summary>`, tests intouchés, `mise run <task>` vert avant et après), puis le signalement `UNTESTED` repris mot pour mot de l'ancienne étape (3).
- Supprimer la ligne « Dispatch says `no measured removal`… ».
- Ligne « Reply » : `DONE <sha of the feat commit>` ou `DONE <sha of the feat commit> <sha of the refactor commit>` ; plus de `NOTHING TO REDUCE`.

- [ ] **Step 2: `SKILL.md`, étape 4**

- Titre : « Green gate, in two steps. » (1) l'implementer répond `DONE <feat sha>` ou `DONE <feat sha> <refactor sha>`, puis ses lignes `UNTESTED` ; les phrases sur le jugement des `UNTESTED` et sur le test ajouté après coup restent. (2) mutant-writer sur le dernier commit de la tâche (le commit `refactor` s'il existe).
- Commande `green` : retirer `[--unneeded-ok <file:start-end>]`.
- « When the slug has several acceptance commits, `--test` is the LAST one » : s'arrêter là (plus de `--since`, plus de PRV-013).
- Supprimer la phrase sur la spec dans la plage `--since` (PRV-014).
- « `--feat` is the top of the branch being judged, checked out: `green` judges the working tree. » : s'arrêter là.
- `--proof-exempt` : « only after a ledgered `Ruling:` » ; plus de `--unneeded-ok`.
- Liste des lignes de sortie : retirer `UNNEEDED <file>:<start>-<end>`.

- [ ] **Step 3: `SKILL.md`, étapes 4b et 8, et les deux autres mentions**

- 4b : le changement neutre ne tourne pas à chaque tâche ; le dispatch du mutant-writer dit `no neutral change` et `green` reçoit `--no-neutral` (`NEUTRAL SKIPPED`). Retirer `no measured removal`, `--no-minimal`, `MINIMAL SKIPPED` et la phrase « The minimal-code proof runs once… ». Le texte sur `--jobs` et sur « preuves complètes » reste ; « preuves complètes » veut dire `green` sans `--no-neutral`.
- 8 : supprimer de « Minimal code, once for the stack » jusqu'à « (`--unneeded-ok`). » ; l'étape commence par la revue finale ; le texte sur `replay` reste.
- « Work without a test », correctif non comportemental : « green with `--proofs <dir>` (the proofs are replayed) », sans « `--since` and `--spec` as in step 4 ».
- « Rebase and conflicts » : « re-run `harness gate green` on each moved branch with its new shas », sans la parenthèse sur `--since`.

- [ ] **Step 4: `regles-generiques.md`**

- Lignes dont la colonne « Tenue par » cite `UNNEEDED`, `--no-minimal` ou le refactor par retrait (registre tâche 4 ; demande du 10/11 « étape refactor » ; PRV-013 ; PRV-014 ; pile MR 25 à 54) : la colonne devient « Retiré le 13/11 : preuve trop lente (`specs/2026-11-13-filets-de-refactoring-design.md`) », suivie de ce qui reste (texte `implementer.md` : plus petit code, `UNTESTED` ; pour la pile MR 25 à 54 : `--no-neutral`, PRV-017).
- Ligne « tâche 12 quater » : garder `--proof-exempt`, retirer la mention `UNNEEDED`.
- Ajouter une ligne : cas réel « Essai du 13/11 : porte très lente, la suite relancée une fois par bloc ajouté » ; règle « La durée de la porte suit le nombre de clauses, jamais la quantité de code ajouté » ; tenue par « Contrôle : pas de ligne `UNNEEDED` (PRV-020) ».

- [ ] **Step 5: spec du 10/11**

Une ligne en tête du paragraphe « Code minimal (décision du 10/11, option 2) » et une en tête de « Rouge, vert, refactor (demande du 10/11) » : `> Retiré le 13/11, voir 2026-11-13-filets-de-refactoring-design.md.` Rien d'autre n'est réécrit.

- [ ] **Step 6: Chercher les mentions restantes**

Run: `grep -rnE 'UNNEEDED|no-minimal|unneeded-ok|--since|MINIMAL SKIPPED|measured removal|NOTHING TO REDUCE' claude harness README.md .config config.toml`
Expected: seules sorties : les deux specs (`2026-11-10-…`, `2026-11-13-…`), les lignes « Retiré le 13/11 » de `regles-generiques.md`, l'entrée PRV-020 de `claude/TESTS.md` et les tests `TestPRV020_*`.

- [ ] **Step 7:** `mise run check` vert, puis **Commit** `docs(sans-code-minimal): textes sans la preuve par retrait`

---

## Partie B — skills, gabarits et socle (branche `docs/stack-skills`, depuis `feat/sans-code-minimal`)

Préalable commun : `docker version` et `docker compose version` répondent (Compose ≥ 2.22) ; sinon s'arrêter et le dire. Forme des deux applications d'essai : une API HTTP `GET /healthz`, `PUT /items/{id}`, `GET /items/{id}`, stockage PostgreSQL ; réglages `PORT` (défaut 8080) et `DATABASE_URL` (obligatoire) ; couches de `hexagonal-go` (`cmd/api`, `internal/domain`, `internal/app`, `internal/adapters/{http,postgres}`) et de `hexagonal-py` (`src/essai/{domain,app,adapters,entrypoints}`, `composition.py`). `<essais>/NOTES.md` reçoit chaque commande lancée, sa sortie utile et la version de l'outil.

### Task 3: Tests d'architecture, socle `architecture`, skills hexagonaux

**Files:**
- Create: `claude/skills/stacked-tdd-mr/templates/socle-application.md` (section `architecture`)
- Modify: `claude/skills/hexagonal-go/SKILL.md`, `claude/skills/hexagonal-py/SKILL.md`, `claude/agents/acceptance-writer.md` (catalogue)

**Interfaces:**
- Produces: IDs `ARC-001` à `ARC-003` ; noms de test `TestARC00n_…` (Go), `test_ARC00n_…` (Python) ; fichier du test d'architecture `tests/architecture/` dans le dépôt cible.

- [ ] **Step 1: Créer les deux applications d'essai** (couches seules, sans Docker), avec `test:go` / `test:py` qui écrivent `.tdd/junit.xml`.

- [ ] **Step 2: Écrire le test d'architecture de chaque application**, dans `tests/architecture/`. Exigences (spec) : liste blanche par couche ; fermeture transitive ; insensible aux alias ; un test par critère, nommé avec l'ID ; la liste des bibliothèques permises au domaine est une constante du fichier de test. Pistes : Go, lecture de `go list -deps -json` (champ `Standard`) depuis la racine du module ; Python, `grimp` sur le graphe d'imports avec les paquets externes. L'outil reste au choix si une piste ne tient pas.

- [ ] **Step 3: Prouver chaque test par ses mutants**, un à la fois, sur chaque application ; noter commande et sortie

| Mutant | Changement | Attendu |
|---|---|---|
| M1 | le domaine importe un adapter | ARC-001 échoue |
| M2 | le même import sous un alias | ARC-001 échoue |
| M3 | le domaine importe un paquet intermédiaire qui importe un adapter | ARC-001 échoue |
| M4 | le domaine importe une bibliothèque tierce hors liste | ARC-001 échoue |
| M5 | un cas d'usage importe un adapter | ARC-002 échoue |
| M6 | un adapter en importe un autre | ARC-003 échoue |
| N1 | un paquet de plus dans le domaine, un adapter de plus câblé à la racine de composition | tout passe |

- [ ] **Step 4: Écrire la section `architecture` du socle**

```markdown
# Socle d'une application Go ou Python

Critères que toute application porte. À recopier dans la spec du dépôt ; l'acceptance-writer les transcrit dans `TESTS.md`. Les IDs sont réservés.

## architecture — Les couches hexagonales ne se mélangent pas

### ARC-001 — Le domaine ne dépend que de lui-même
- **Étant donné** le code du dépôt
- **Quand** le test d'architecture est lancé
- **Alors** aucun paquet du domaine n'importe, directement ou par un intermédiaire, un paquet de `app`, un adapter, un point d'entrée, ni une bibliothèque hors de la liste que le fichier de test énumère

### ARC-002 — Les cas d'usage ne connaissent aucune technologie
- **Étant donné** le code du dépôt
- **Quand** le test d'architecture est lancé
- **Alors** aucun paquet de `app` n'importe, directement ou par un intermédiaire, un adapter ni un point d'entrée

### ARC-003 — Seule la racine de composition assemble
- **Étant donné** le code du dépôt
- **Quand** le test d'architecture est lancé
- **Alors** un adapter n'est importé que par la racine de composition
```

Suivi d'une note : mutants types (M1 à M6), changement neutre type (N1), « Cas réel » à remplir par dépôt.

- [ ] **Step 5: `hexagonal-go`**

- Nouvelle section « SOLID, in Go terms », cinq puces, pendant de celle de `hexagonal-py` : SRP (un paquet, une raison de changer ; un cas d'usage fait une chose) ; OCP (ajouter un adapter ou envelopper une interface, sans modifier les appelants) ; LSP (un adapter tient le contrat documenté de l'interface : erreurs, ordre) ; ISP (interfaces de 1 à 3 méthodes déclarées chez le consommateur) ; DIP (les cas d'usage dépendent d'interfaces injectées, jamais d'un adapter).
- « Enforcing the layers » : le test d'architecture d'abord (critères `ARC-…` du socle, où il vit, ce qu'il attrape : alias et imports transitifs, « verified » avec les sorties de M2 et M3) ; forbidigo reste, présenté comme retour rapide au commit.
- Une ligne de renvoi vers `twelve-factor` et `compose-dev` pour une application avec `Dockerfile`.

- [ ] **Step 6: `hexagonal-py`** : même paragraphe sur le test d'architecture (la limite « ruff does not follow transitive imports » renvoie au test) et même renvoi.

- [ ] **Step 7: Catalogue de `acceptance-writer.md`** : ligne « Architecture (Go, Python) » — motif (test sous `tests/architecture/`, un test par critère `ARC-…`, liste permise dans le fichier), interdit (asserter sur le texte d'un fichier de configuration de lint ; lister les paquets un par un au lieu d'une règle par couche), statut « proven <date> » avec le nombre de mutants attrapés.

- [ ] **Step 8:** `mise run check` vert, puis **Commit** `docs(stack-skills): tests d'architecture et SOLID`

### Task 4: `compose-dev`, gabarits compose, section applicative de `repo-tooling`

**Files:**
- Create: `claude/skills/compose-dev/SKILL.md`, `claude/skills/repo-tooling/templates/compose.go.yaml`, `claude/skills/repo-tooling/templates/compose.py.yaml`
- Modify: `claude/skills/repo-tooling/SKILL.md`, `config.toml` (`[dotfiles]`)

**Interfaces:**
- Consumes: les applications d'essai de la tâche 3.
- Produces: tâches `dev`, `lint:compose` ; convention de lancement des tests : `docker compose -p <nom propre à l'exécution> up -d --wait`, port d'hôte éphémère lu par `docker compose port`, puis `down -v`. La tâche 5 s'en sert.

- [ ] **Step 1: Ajouter `Dockerfile` (multi-étapes, non root) et `compose.yaml` aux deux applications** : service `app` construit depuis le `Dockerfile` de production, `read_only: true`, réglages par `environment`, port `"${APP_PORT:-8080}:8080"` ; service `db` PostgreSQL avec `healthcheck` ; `depends_on: db: condition: service_healthy` ; section `develop.watch`.

- [ ] **Step 2: Vérifier et noter**

| Vérification | Commande ou geste | Ce qu'on note |
|---|---|---|
| V1 | `docker compose version` | version ; watch disponible |
| V2 | Go : `docker compose up --watch`, modifier la réponse d'un handler | l'action `rebuild` recharge ; durée |
| V3 | Python : action `sync+restart` avec `read_only: true` | marche ou non ; si non, `rebuild` aussi en Python |
| V4 | deux `up -d --wait` simultanés, noms de projet distincts, `APP_PORT=0` | pas de collision ; `docker compose port app 8080` donne chaque port |
| V5 | `up -d --wait` avec une base qui ne démarre pas | code de sortie non nul |
| V6 | `docker compose config --quiet` sur un fichier invalide | code de sortie non nul |
| V7 | origine de `docker-build-strategies` (`skill.yaml`, licence) | le skill public `docker-compose-patterns` existe ou non |

- [ ] **Step 3: Écrire `compose-dev/SKILL.md`** — description : « Use when adding or reviewing the compose.yaml of a Go or Python application, its watch loop, or the tests that start the product through Compose. » Contenu : règles du service applicatif (construit le `Dockerfile` de production, `read_only`, non root, réglages par `environment`) ; services d'appui (`healthcheck`, `depends_on` conditionnel) ; `develop.watch` par langage, d'après V2 et V3 ; lancement par les tests, d'après V4 et V5 ; ce que le skill ne couvre pas (renvoi à `docker-build-strategies`, `dockerfile-restricted`). Si V7 trouve le skill public : le reprendre tel quel sous `claude/skills/docker-compose-patterns/` avec sa licence et sa ligne `[dotfiles]`, et dire en tête de `compose-dev` qu'il le complète.

- [ ] **Step 4: Écrire les deux gabarits** d'après les fichiers vérifiés, sans rien de propre à l'application d'essai.

- [ ] **Step 5: `repo-tooling/SKILL.md`** — nouvelle section « Go or Python application (repo with a Dockerfile) » : ne concerne pas l'infrastructure as code ; `compose.yaml` avec watch (gabarits) ; tâches `dev` (`docker compose up --watch`, hors `check`), `lint:compose` (`docker compose config --quiet`), `test:container`, et `test:api` / `test:ui` quand le dépôt a une API ou un front ; un rapport par tâche de test sous `.tdd/` ; ces tâches séparées de `test:go` / `test:py` ; Docker requis sur le poste et en CI, ligne GitHub à vérifier sur le premier dépôt réel, ligne GitLab « unverified ». Ajouter au tableau « By stack » les lignes Compose, API (Bruno), front (Playwright).

- [ ] **Step 6: `config.toml`** : une ligne `[dotfiles]` par skill créé, sur le modèle des lignes voisines, dans l'ordre alphabétique.

- [ ] **Step 7:** `mise run check` vert (dont `lint:harness`), puis **Commit** `docs(stack-skills): compose avec watch`

### Task 5: `twelve-factor`, socle `douze-facteurs`, catalogue

**Files:**
- Create: `claude/skills/twelve-factor/SKILL.md`
- Modify: `claude/skills/stacked-tdd-mr/templates/socle-application.md`, `claude/skills/stacked-tdd-mr/templates/acceptance-criteria.md`, `claude/agents/acceptance-writer.md`, `config.toml`

**Interfaces:**
- Consumes: convention de lancement de la tâche 4 ; tâche `test:container`.
- Produces: IDs `FAC-001` à `FAC-005`.

- [ ] **Step 1: Écrire les cinq tests sur chaque application d'essai** (tests natifs du langage sous `tests/container/`, qui pilotent `docker compose` ; rapport `.tdd/container.xml` ; tâche `test:container`).

- [ ] **Step 2: Prouver chaque test par un mutant** ; noter commande et sortie

| Critère | Mutant | Attendu |
|---|---|---|
| FAC-001 | port écrit en dur dans le code | échoue |
| FAC-002 | valeur par défaut donnée à `DATABASE_URL` | échoue |
| FAC-003 | log de requête retiré, puis écrit dans un fichier | échoue les deux fois |
| FAC-004 | `CMD` en forme shell (le signal n'atteint pas le processus) | échoue (code 137) |
| FAC-005 | `read_only: true` retiré du compose | échoue |

- [ ] **Step 3: Ajouter la section `douze-facteurs` au socle**

```markdown
## douze-facteurs — L'image se comporte en application à douze facteurs

Dépôt avec `Dockerfile`. Le produit est l'image, lancée par le `compose.yaml` du dépôt.

### FAC-001 — Le port d'écoute vient de l'environnement
- **Étant donné** l'application lancée par le compose avec un port d'écoute autre que celui par défaut
- **Quand** une requête est envoyée sur ce port
- **Alors** l'application répond

### FAC-002 — Une variable obligatoire absente arrête le démarrage
- **Étant donné** l'application lancée par le compose sans une variable d'environnement obligatoire
- **Quand** le conteneur démarre
- **Alors**
  - il sort avec un code non nul
  - sa sortie nomme la variable manquante

### FAC-003 — Les logs sortent sur la sortie standard
- **Étant donné** l'application lancée par le compose
- **Quand** une requête est servie
- **Alors** la sortie du conteneur porte la ligne de log de cette requête

### FAC-004 — L'arrêt est propre
- **Étant donné** l'application lancée par le compose
- **Quand** le conteneur reçoit SIGTERM
- **Alors** il sort avec le code 0 avant la fin du délai d'arrêt

### FAC-005 — Aucun état local
- **Étant donné** l'application lancée par le compose
- **Quand** elle sert son cas nominal
- **Alors**
  - le système de fichiers racine du conteneur est en lecture seule
  - le processus ne tourne pas en root
```

Suivi d'une note : un critère sans objet (FAC-001 pour un worker sans port) s'écarte par un `Ruling:` consigné ; mutants types de l'étape 2.

- [ ] **Step 4: Écrire `twelve-factor/SKILL.md`** — description : « Use when writing or reviewing a Go or Python application that ships as a container image, or when it reads configuration, logs, handles shutdown or keeps state. » Un tableau des douze facteurs : ce que le code fait, en termes Go, Python et Kubernetes, et qui le tient. I : un dépôt, une application (règle écrite). II : `repo-tooling`. III : `FAC-001`, `FAC-002`. IV : service d'appui donné par une URL de l'environnement (règle écrite, `compose-dev`). V : image construite une fois, réglée au lancement (règle écrite). VI : `FAC-005`. VII : `FAC-001`. VIII : montée en charge par réplicas, pas de démon interne (règle écrite). IX : `FAC-004`. X : le compose construit le `Dockerfile` de production (`compose-dev`). XI : `FAC-003`. XII : tâche d'administration lancée comme commande de la même image (règle écrite).

- [ ] **Step 5: Catalogue de `acceptance-writer.md`**

- Lignes « HTTP API » et « Web front » : le produit est démarré par le compose du dépôt, par l'aide commune des tests (convention de la tâche 4), à la place de « the service the mise task starts and stops » et de la fixture de session ; statut mis à jour d'après les notes.
- Nouvelle ligne « Running container (twelve-factor) » : motif (tests natifs sous `tests/container/`, compose, un test par critère `FAC-…`), interdit (asserter sur le texte du `Dockerfile` ou du `compose.yaml`, attente fixe), statut « proven <date> ».

- [ ] **Step 6: `templates/acceptance-criteria.md`** : la ligne « Chaque type de livrable a son outil de test » gagne « architecture → test d'architecture ; conteneur en exécution → tests natifs sur le compose », et une phrase renvoie au socle pour une application Go ou Python.

- [ ] **Step 7:** ligne `[dotfiles]` de `twelve-factor` ; `mise run check` vert ; **Commit** `docs(stack-skills): douze facteurs`

### Task 6: Détection et skills de stack

**Files:**
- Create: `claude/skills/repo-stack/SKILL.md`, `claude/skills/stack-go-app/SKILL.md`, `claude/skills/stack-py-app/SKILL.md`
- Modify: `claude/skills/stacked-tdd-mr/SKILL.md` (Setup, étape 8), `claude/docs/regles-generiques.md`, `config.toml`

**Interfaces:**
- Consumes: tous les skills et le socle des tâches 3 à 5.
- Produces: table de détection (ci-dessous), lue par `stacked-tdd-mr`.

- [ ] **Step 1: `repo-stack/SKILL.md`** — description : « Use when starting work in a repository that holds Go or Python code, before writing or reviewing code or planning a stack: detects the repository's stack and names the stack skill to follow. » Contenu : regarder la racine du dépôt, appliquer la table, charger le skill nommé avant toute autre chose ; ne décrit aucune stack lui-même.

| À la racine | Skill | Ajout avec un `Dockerfile` |
|---|---|---|
| `go.mod` et un dossier `cmd/` ou `internal/` | `stack-go-app` | socle `douze-facteurs`, `compose-dev` |
| `pyproject.toml` et un dossier `src/` | `stack-py-app` | socle `douze-facteurs`, `compose-dev` |
| rien de cela | aucun : le dire et n'imposer rien | — |

Limites à écrire dans le skill : code dans un sous-dossier, bibliothèque Go dont les paquets sont à la racine ; dans ces cas, demander à l'utilisateur.

- [ ] **Step 2: `stack-go-app/SKILL.md`** — description : « Use when repo-stack names it: the stack required for a Go application. » Une liste, une ligne par exigence, chacune avec le skill qui la détaille et rien de plus : architecture hexagonale et SOLID → `hexagonal-go` ; outillage → `repo-tooling` ; socle `architecture` (`ARC-001` à `ARC-003`) → `stacked-tdd-mr/templates/socle-application.md` ; avec un `Dockerfile` : douze facteurs → `twelve-factor` et socle `douze-facteurs` (`FAC-001` à `FAC-005`), compose avec watch → `compose-dev`, image → `docker-build-strategies` et `dockerfile-restricted` ; API HTTP → Bruno ; front → Playwright (catalogue de `acceptance-writer`). Dernière ligne : le plus petit code qui fait passer les tests, rien pour un cas hypothétique.

- [ ] **Step 3: `stack-py-app/SKILL.md`** : la même liste avec `hexagonal-py`.

- [ ] **Step 4: Vérifier la table sur cinq dispositions de dossier jetables** ; noter le résultat de chacune

| Cas | Racine | Attendu |
|---|---|---|
| D1 | `go.mod`, `internal/`, `Dockerfile` | `stack-go-app` avec les ajouts |
| D2 | `pyproject.toml`, `src/` | `stack-py-app` sans les ajouts |
| D3 | `Chart.yaml`, `pyproject.toml`, `tests/` | aucun |
| D4 | `main.tf`, `go.mod`, `test/` | aucun |
| D5 | `go.mod`, fichiers `.go` à la racine | aucun, limite signalée |

- [ ] **Step 5: `stacked-tdd-mr/SKILL.md`**

- Setup, nouveau point après le point 3 : appeler `repo-stack` et suivre le skill de stack qu'il nomme ; un dépôt qui a une stack et pas son socle commence sa pile par les tâches du socle, après `chore(tooling)` (`Mode: characterize` si le code est déjà conforme, `atdd` sinon) ; avant d'installer le socle, vérifier qu'aucune entrée de `TESTS.md` ne porte déjà un ID `ARC-…` ou `FAC-…` pour autre chose, sinon s'arrêter et le dire.
- Étape 8 : le brief de la revue finale nomme le skill hexagonal du langage quand le dépôt a une stack ; une remarque SOLID cite le code visé et passe par « Work without a test » (commit `refactor`, `green` rejouée avec les preuves).

- [ ] **Step 6: `regles-generiques.md`** : trois lignes. Demande du 13/11 « tests d'architecture » → un dépôt d'application porte un socle de critères d'architecture, prouvés par mutants (texte : `socle-application.md`, `stack-*-app`). Demande du 13/11 « douze facteurs » → ce qui se teste sur le conteneur est un critère du socle, le reste une règle écrite (texte : `twelve-factor`). Demande du 13/11 « un skill qui détecte le dépôt et impose le skill de la stack » → la stack voulue est écrite une fois par type de dépôt (texte : `repo-stack` ; aucun contrôle de porte, D10).

- [ ] **Step 7:** lignes `[dotfiles]` des trois skills ; `mise run check` vert ; **Commit** `docs(stack-skills): détection du dépôt et skills de stack`

- [ ] **Step 8: Remise** : résumé pour l'utilisateur avec, par skill, les affirmations « verified » et celles restées « unverified » (au moins la ligne GitLab), et le rappel `mise bootstrap --update` après fusion pour recompiler `~/.local/bin/harness` et poser les liens des nouveaux skills.
