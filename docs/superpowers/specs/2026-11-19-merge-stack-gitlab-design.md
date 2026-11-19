# `harness merge-stack` sur GitLab

## Intention

`harness merge-stack <MR du haut>` fusionne une pile de merge requests en commençant par le bas, puis prouve que la branche visée par la pile porte le code du haut et que son pipeline passe ; au moindre doute il s'arrête sur `FAIL`. Il ne parle aujourd'hui qu'à `gh`. Le 16/11, un `--dry-run` sur une MR GitLab a échoué (`FAIL default branch: gh repo: none of the git remotes … point to a known GitHub host`) ; depuis, les fusions GitLab passent par un script API figé sur le sha, lancé à la main. Le but : la même commande, le même comportement et les mêmes preuves sur GitLab, pour retirer ce contournement.

Premier usage réel : un lot de MR indépendantes réparties sur une vingtaine de projets, chacun avec une MR vers `dev` et une vers `main`. Chaque MR y est une pile d'une seule MR, lancée depuis le clone de son projet.

## Objectifs

1. Un adaptateur `gitlab` du port `Forge` (`harness/internal/app/mergestack.go`), basé sur `glab api`.
2. Le choix de l'adaptateur selon l'hôte de `origin`, tel que le déclare le fichier `forges` du répertoire local (`CLAUDE_LOCAL_DIR`) ; un hôte inconnu donne un `FAIL` explicite.
3. Une pile peut viser une autre branche que la branche par défaut (`dev`), sur les deux forges ; la preuve porte sur la branche visée.
4. Un faux `glab` pour les tests bats, et les scénarios de `merge-stack.bats` rejoués sur GitLab.
5. La skill `forge-glab`, la règle globale (`claude/CLAUDE.md`) et `stacked-tdd-mr` disent d'utiliser `harness merge-stack` sur GitLab ; la phrase « On GitLab the command does not exist yet » disparaît.

## Non-objectifs

- Un mode « lot » qui fusionne des MR de plusieurs projets en un appel : une pile = un projet = un clone courant, comme sur GitHub.
- Les MR venant d'un fork, les MR d'un autre projet que celui de `origin`.
- Les règles d'approbation GitLab : si GitLab refuse (`not_approved`, `discussions_not_resolved`…), on s'arrête et on le dit, on ne contourne rien.
- `glab stack` (expérimental).

## Anti-objectifs

- Un `PASS` alors que le code fusionné n'est pas celui relu, ou qu'il atterrit sur une branche abandonnée.
- Une fusion qui part sur un sha autre que celui vérifié.
- Le moindre hôte interne ou nom de dépôt réel dans le code, les tests, la spec ou la PR : le dépôt est public. Les tests utilisent `forge.example.test`.

## Contraintes

- Dépôt public `Wihrt/mise` ; passage par PR, CI verte.
- Architecture hexagonale existante : le cœur (`app.MergeStack`) ne connaît que les ports `Forge`, `History`, `Validator`.
- L'adaptateur passe par `glab api` (authentification, proxy et `:id` résolu depuis `origin` sont gérés par glab).
- Le hook « jamais de fusion sans demande » (`claude/hooks/tests/never-merge-stack.bats`, FUS-009) continue de s'appliquer tel quel.

## Approches

1. **Retenue — un adaptateur `gitlab` à côté de `github`, derrière le port existant.** `main.go` choisit l'adaptateur par l'hôte de `origin`. Le cœur ne change que sur la fin de la pile (voir plus bas).
2. Écartée — appeler l'API REST de GitLab directement en HTTP : il faudrait refaire l'authentification, le proxy et la résolution du projet que glab fait déjà.
3. Écartée — envelopper le script de fusion actuel : il ne prouve rien après la fusion.

Pas d'ADR (décision de l'orchestrateur) : la correspondance hôte → outil existe déjà (`harness/internal/app/local/local.go`, `harness local forge`, défauts `github.com → gh`, `gitlab.com → glab`) ; `merge-stack` la réutilise, il n'y a pas de nouvelle décision d'architecture. Un second adaptateur derrière un port existant est le cas prévu par le port.

## Ce qui a été décidé

### Fin de la pile (les deux forges)

Aujourd'hui `resolve` descend de MR en MR jusqu'à la branche par défaut et refuse toute autre cible (`#N targets X, which is neither main nor the branch of an open merge request`). Désormais :

- La pile s'arrête sur la première branche qui n'est la tête d'aucune MR **ouverte**. Cette branche est **la cible de la pile** ; elle remplace la branche par défaut partout : contrôle `OUTDATED`, repointage des MR du dessus, preuve de l'arbre, pipeline après fusion.
- **Garde** : si la cible est ou a été la branche source d'une MR, quel que soit son état (fermée, fusionnée), c'est un `FAIL` qui nomme la branche et cette MR ; rien n'est fusionné. Cela bloque le cas A ← B où A a été fermée sans fusion et sa branche survit : sans garde, B serait fusionnée dans la branche abandonnée de A et le résultat serait `PASS`. `dev` et `main` passent.
- Les piles qui visent la branche par défaut se comportent exactement comme avant.

### Choix de la forge

- `merge-stack` lit l'hôte de `origin` du clone courant et cherche l'outil comme `harness local forge` (fichier `forges` du répertoire local, puis les défauts publics). `gh` → adaptateur `github`, `glab` → adaptateur `gitlab`.
- Hôte absent des deux : `FAIL` qui nomme l'hôte et le fichier `forges` à compléter ; aucun appel à `gh` ni `glab`, rien n'est fusionné.

### Adaptateur `gitlab`

Tous les appels : `glab api` dans le clone courant, projet `:id`.

- **Branche par défaut** : `projects/:id`, champ `default_branch`.
- **MR lue** (`Get`) : référence acceptée = numéro (`12`), `!12`, URL web de MR du projet (`…/-/merge_requests/12`), ou nom de branche source (renvoie à la recherche par branche). Lecture `projects/:id/merge_requests/<iid>`. Correspondance : `iid` → numéro, `web_url`, `source_branch`, `target_branch`, `sha`, `state == "opened"` → ouverte, `draft`. Une URL d'un autre projet que `origin` est refusée.
- **Prête à fusionner** : en plus des contrôles du cœur (ouverte, pas brouillon, empilée sur la branche du dessous, pipeline), une MR n'est pas prête si `has_conflicts` est vrai, ou si `detailed_merge_status` ne vaut pas `mergeable` ; le message nomme la valeur reçue. Ce contrôle est fait avant la première fusion pour toute la pile (rien n'est fusionné si une MR ne l'est pas).
- **MR par branche** (`ByHead`) : `projects/:id/merge_requests?source_branch=<b>&state=opened`. La garde de fin de pile cherche la même branche dans tous les états.
- **Repointage** : `PUT projects/:id/merge_requests/<iid>` avec `target_branch=<cible>`. GitLab repointe lui-même les MR du dessus quand la MR du bas est fusionnée avec suppression de sa branche source (au plus quatre MR, et seulement si la suppression réussit — [doc GitLab](https://docs.gitlab.com/user/project/merge_requests/#update-merge-requests-when-target-branch-merges)) ; le cœur garde son repointage explicite, sans effet quand GitLab l'a déjà fait, pour couvrir les autres cas.
- **Checks d'une MR** : `head_pipeline` de la MR, attendu jusqu'à sa fin (mêmes délais que GitHub : `HARNESS_POLL_INTERVAL`, `HARNESS_CHECKS_GRACE`, `HARNESS_CHECKS_TIMEOUT`). Vert seulement si `head_pipeline.sha` est le sha de tête et `status == success` ; si le projet lance des pipelines de MR (une source `merge_request_event` dans `projects/:id/merge_requests/<iid>/pipelines`), la source doit aussi être `merge_request_event`. Pas de `head_pipeline` après le délai de grâce : « pas de pipeline », comme `gh` sans checks.
- **Fusion** : attendre que `detailed_merge_status` sorte des états transitoires (`checking`, `unchecked`, `preparing`…) après un repointage ; `mergeable` → `PUT projects/:id/merge_requests/<iid>/merge` avec `sha=<sha vérifié>` et `should_remove_source_branch=true` ; toute autre valeur, un refus de GitLab (sha qui a bougé) ou un état qui n'est pas `merged` après l'appel → erreur qui nomme la cause, donc `STOPPED` dans le cœur.
- **Checks d'un commit** (après fusion) : pipelines du sha sur la branche cible (`projects/:id/pipelines?sha=<sha>`), attendus jusqu'à leur fin ; aucun après le délai de grâce → « pas de pipeline » (la commande `--validate` fait alors foi, comme aujourd'hui).

### Sortie

Les lignes de sortie restent celles de GitHub (`STACK`, `READY`, `MERGED #n <url>`, `STOPPED`, `TREE IDENTICAL`, `CHECKS PASS`, `PASS`, `FAIL …`), avec `#n` pour le numéro sur les deux forges (décision de l'orchestrateur : un seul contrat de sortie). Là où la sortie nommait la branche par défaut, elle nomme la cible de la pile.

### Tests

- Faux `glab` dans `claude/hooks/tests/fake-forge/glab`, sur le modèle du faux `gh` : état des MR dans un JSON, dépôt dans un clone nu, `glab api` avec `-X`, `-f`/`-F`, chemins `projects/:id/...`. La fusion fait un vrai commit de fusion dans le clone nu, supprime la branche source et repointe les MR qui la visaient. Il enregistre les paramètres reçus (sha, suppression de branche) pour que les tests les vérifient.
- Les scénarios FUS-001 à FUS-008 de `merge-stack.bats` rejoués sur le faux `glab`, `origin` sur `forge.example.test` déclaré `glab` dans un `forges` de test.

### Documentation

- `claude/CLAUDE.md`, section « Merge requests » : retirer « On GitLab the command does not exist yet: hand over the MR links. »
- `claude/skills/forge-glab/SKILL.md` : comme `forge-gh`, « When the user asks for the merge: `harness merge-stack <top MR>`, never `glab mr merge` ».
- `claude/skills/stacked-tdd-mr/SKILL.md`, « Merging a stack » : retirer la mention GitLab, s'il y en a une.

### Points tranchés après relecture (décisions de l'orchestrateur)

- **Pas de nouveau contrôle du pipeline après repointage**, comme sur GitHub aujourd'hui : la fusion est figée sur le sha relu, et la preuve après fusion vérifie le pipeline de la cible. Dans `Merge`, l'adaptateur `gitlab` traite `ci_still_running` comme un état transitoire (il attend), au même titre que `checking`/`unchecked`/`preparing`.
- **Adaptateur `github` inchangé** pour son hôte : `gh` résout lui-même l'hôte (GitHub Enterprise compris via `gh auth`). Le fichier `forges` ne sert qu'à choisir l'adaptateur ; ce choix se fait avant tout appel à `gh` ou `glab` (FGL-010).
- **Garde de fin de pile, fermée en cas de doute** : GitLab `merge_requests?source_branch=<b>` sans filtre d'état, toutes les pages ; GitHub `gh pr list --head <b> --state all` avec une limite assez haute, en ne comptant que les PR du dépôt lui-même (pas des forks). Requête en erreur → `FAIL`, rien n'est fusionné. Une MR **fusionnée** dont la branche source est la cible compte aussi comme refus : c'est le cas d'une pile à moitié fusionnée dont la branche n'a pas été supprimée, déjà refusé aujourd'hui ; le message dit de repointer la MR suivante.
- **MR sans pipeline** : comportement actuel conservé — « pas de pipeline » ne bloque pas la MR avant fusion (seuls un pipeline rouge ou une erreur bloquent, `ready` dans `mergestack.go`) ; la preuve après fusion exige alors un pipeline de la cible ou `--validate`.

## Laissé au constructeur

- Forme exacte des changements de port (méthode pour chercher une branche source dans tous les états ; `CommitChecks` reçoit-il la branche cible).
- Découpage du code de lecture de `forges` pour que `merge-stack` le réutilise sans dupliquer.
- Analyse des références de MR et des URL GitLab.

## Hors périmètre, après la livraison

- Mettre à jour la mémoire `fusion-gitlab-script-utilisateur` : `harness merge-stack` remplace le script.
- Lot CNAME : une commande `harness merge-stack <iid>` par MR, depuis le clone de son projet.

## Critères d'acceptation

Section `TESTS.md` : `fusion-gitlab`. Tous les tests passent par le binaire `harness merge-stack`, avec les faux `gh`/`glab` sur le `PATH` et un `forges` de test.

### FGL-001 — Une pile GitLab se fusionne du bas vers le haut et la cible porte le code du haut
- **Étant donné** un dépôt dont `origin` est sur `forge.example.test`, déclaré `glab` dans `forges` ; une pile !1 ← !2 ← !3 sur `main`, ouvertes, vertes, `mergeable`
- **Quand** `harness merge-stack 3`
- **Alors**
  - la sortie contient `MERGED #1`, `MERGED #2`, `MERGED #3` dans cet ordre
  - les trois MR sont à l'état `merged`
  - l'arbre de `main` sur le dépôt est celui de la tête de !3
  - la sortie se termine par `PASS` et le code de sortie est 0
- **Frontière** : binaire `harness`, faux `glab`
- **Remplace** : —

### FGL-002 — Une MR GitLab seule se fusionne sans celles du dessus
- **Étant donné** la pile de FGL-001
- **Quand** `harness merge-stack 1`
- **Alors**
  - !1 est `merged`
  - !2 et !3 restent `opened`
  - la sortie se termine par `PASS`
- **Frontière** : binaire `harness`, faux `glab`

### FGL-003 — Rien n'est fusionné tant qu'une MR GitLab n'est pas prête
- **Étant donné** la pile de FGL-001, avec sur !2, dans un cas chacun : pipeline `failed` ; `head_pipeline` sur un autre sha que le sha de tête ; `has_conflicts` vrai ; `detailed_merge_status` à `not_approved` ; `main` qui a avancé sous la pile
- **Quand** `harness merge-stack 3`
- **Alors**
  - aucune MR n'est fusionnée
  - la sortie nomme !2 et la cause (`CHECKS FAIL #2`, `CONFLICTS #2`, `NOT MERGEABLE #2: not_approved`, `OUTDATED #1`)
  - la sortie se termine par `FAIL nothing merged` et le code de sortie est 1
- **Frontière** : binaire `harness`, faux `glab`

### FGL-004 — La fusion GitLab est figée sur le sha vérifié et supprime la branche source
- **Étant donné** la pile de FGL-001
- **Quand** `harness merge-stack 3`
- **Alors**
  - chaque appel de fusion reçu par le faux `glab` porte `sha` égal au sha de tête de la MR lu avant la fusion
  - chaque appel de fusion porte `should_remove_source_branch=true`
- **Frontière** : binaire `harness`, faux `glab`

### FGL-005 — La fusion GitLab s'arrête à la MR qui résiste et dit ce qui reste
- **Étant donné** la pile de FGL-001, et le faux `glab` refuse la fusion de !2 (tête qui a bougé)
- **Quand** `harness merge-stack 3`
- **Alors**
  - la sortie contient `STOPPED #2`, `MERGED: #1`, `REMAINING: #2 #3`
  - la sortie se termine par `FAIL stack partly merged` et le code de sortie est 1
  - !1 est `merged`, !2 et !3 restent `opened`
- **Frontière** : binaire `harness`, faux `glab`

### FGL-006 — Après fusion GitLab, une cible qui n'est pas le code du haut est signalée
- **Étant donné** la pile de FGL-001, et un commit poussé sur `main` par un tiers pendant la fusion
- **Quand** `harness merge-stack 3`
- **Alors**
  - la sortie contient `TREE DIFFERS`
  - la sortie se termine par `FAIL stack merged, main differs from the top of the stack`
- **Frontière** : binaire `harness`, faux `glab`

### FGL-007 — Après fusion GitLab, une cible rouge est signalée
- **Étant donné** la pile de FGL-001, et le pipeline du commit de fusion sur `main` `failed`
- **Quand** `harness merge-stack 3`
- **Alors**
  - la sortie contient `CHECKS FAIL main`
  - la sortie se termine par `FAIL stack merged, main is red` et le code de sortie est 1
- **Frontière** : binaire `harness`, faux `glab`

### FGL-008 — Sans pipeline GitLab, la commande de validation fait foi
- **Étant donné** la pile de FGL-001, et aucun pipeline sur `main` après fusion
- **Quand** `harness merge-stack --validate <commande> 3`, avec une commande qui réussit, puis une qui échoue, puis sans `--validate`
- **Alors**
  - commande qui réussit : `VALIDATION PASS`, puis `PASS`
  - commande qui échoue : `VALIDATION FAIL`, puis `FAIL stack merged, main is red`
  - sans `--validate` : `CHECKS NONE`, puis `FAIL stack merged, main not verified`
- **Frontière** : binaire `harness`, faux `glab`

### FGL-009 — Un essai à blanc GitLab vérifie la pile sans rien fusionner
- **Étant donné** la pile de FGL-001
- **Quand** `harness merge-stack --dry-run 3`
- **Alors**
  - la sortie contient `READY`
  - aucune MR n'est fusionnée et le faux `glab` n'a reçu aucun appel de fusion ni de repointage
  - la sortie se termine par `PASS nothing merged (dry run)`
- **Frontière** : binaire `harness`, faux `glab`

### FGL-010 — Un hôte de forge inconnu est refusé
- **Étant donné** un dépôt dont `origin` est sur `inconnu.example.test`, absent de `forges`
- **Quand** `harness merge-stack 1`
- **Alors**
  - la sortie se termine par un `FAIL` qui nomme `inconnu.example.test` et le fichier `forges`
  - le code de sortie est 1
  - ni le faux `gh` ni le faux `glab` n'ont été appelés
- **Frontière** : binaire `harness`

### FGL-011 — Une pile qui vise une autre branche que la branche par défaut s'arrête sur elle
- **Étant donné** une MR !1 de `fix` vers `dev`, `dev` n'étant la branche source d'aucune MR et `main` étant la branche par défaut ; une fois avec le faux `glab`, une fois avec le faux `gh`
- **Quand** `harness merge-stack 1`
- **Alors**
  - !1 est fusionnée dans `dev`
  - `main` n'a pas bougé
  - la sortie contient `TREE IDENTICAL dev` et `CHECKS PASS dev`
  - la sortie se termine par `PASS`
- **Frontière** : binaire `harness`, faux `gh` et faux `glab`

### FGL-012 — Une cible qui a été la branche source d'une MR est refusée
- **Étant donné** une pile !1 (`a` vers `main`) ← !2 (`b` vers `a`), où !1 est fermée sans fusion et la branche `a` existe encore ; une fois avec le faux `glab`, une fois avec le faux `gh`
- **Quand** `harness merge-stack 2`
- **Alors**
  - aucune MR n'est fusionnée
  - la sortie se termine par un `FAIL` qui nomme la branche `a` et la MR #1
  - le code de sortie est 1
- **Frontière** : binaire `harness`, faux `gh` et faux `glab`

### FGL-013 — Une MR GitLab se désigne par `!n` ou par son URL, pas par l'URL d'un autre projet
- **Étant donné** la pile de FGL-001
- **Quand** `harness merge-stack --dry-run '!3'`, puis `harness merge-stack --dry-run <web_url de !3>`, puis `harness merge-stack --dry-run <URL de MR d'un autre projet sur forge.example.test>`
- **Alors**
  - avec `!3`, la sortie se termine par `PASS nothing merged (dry run)`
  - avec l'URL de !3, la sortie se termine par `PASS nothing merged (dry run)`
  - avec l'URL d'un autre projet, la sortie se termine par un `FAIL` et le code de sortie est 1
- **Frontière** : binaire `harness`, faux `glab`

### FGL-014 — Sur un projet à pipelines de MR, seul un pipeline de MR vert compte
- **Étant donné** la pile de FGL-001, le projet lance des pipelines de MR (une source `merge_request_event` dans les pipelines de !2), et le `head_pipeline` de !2 est un pipeline `push` en `success` sur le sha de tête
- **Quand** `harness merge-stack 3`
- **Alors**
  - aucune MR n'est fusionnée
  - la sortie nomme !2 (`CHECKS FAIL #2` ou `CHECKS UNKNOWN #2`)
  - la sortie se termine par `FAIL nothing merged`
- **Frontière** : binaire `harness`, faux `glab`

### FGL-015 — Après un repointage, la fusion GitLab attend que GitLab se prononce
- **Étant donné** la pile de FGL-001, et après la fusion de !1 le faux `glab` répond `detailed_merge_status: checking` pour !2 aux deux premières lectures, puis `mergeable`
- **Quand** `harness merge-stack 3`
- **Alors**
  - les trois MR sont `merged`
  - aucun appel de fusion de !2 n'est reçu tant que le statut est `checking`
  - la sortie se termine par `PASS`
- **Frontière** : binaire `harness`, faux `glab`
