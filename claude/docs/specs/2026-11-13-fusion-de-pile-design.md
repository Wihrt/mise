# Fusion d'une pile de merge requests — spec

Date : 2026-11-13. Périmètre : le binaire `harness` (`harness/`), le hook `never-merge`, `claude/CLAUDE.md`, `claude/skills/stacked-tdd-mr/SKILL.md`, `claude/skills/forge-gh/SKILL.md`.

## Objectif

Demande du 13/11 : « une fois que j'ai review une pile de MR, j'ai un peu la flemme de tout merger, donc l'idée, c'est de pouvoir merger une pile et vérifier que tout est OK à la fin. Merger une pile ou une MR d'ailleurs. On a le droit de merge, mais seulement à ma demande, et si on merge, on vérifie que la CI ou les tests de validation passent. »

Aujourd'hui la fusion est interdite sans exception : le hook `never-merge` refuse `gh pr merge` et `glab mr merge`, et l'utilisateur fusionne chaque MR à la main, change la base de la suivante, attend son pipeline, recommence.

## Décisions prises (13/11)

| Sujet | Décision | Par |
|---|---|---|
| Ce qui doit être vrai à la fin | La branche par défaut porte exactement le code du haut de la pile, tel qu'il a été relu. Les états intermédiaires ne comptent pas. | utilisateur |
| Échec | Vérification rouge ou code différent : arrêt, compte rendu, aucune réparation ni retour arrière. | utilisateur |
| Vérification | Le pipeline de la branche par défaut, ou une commande de validation quand il n'y en a pas. | utilisateur |
| Forme de l'accord | Une seule commande, `harness merge-stack`, pour laquelle le hook répond `ask` : l'utilisateur valide une fois, pour cette pile. Un hook ne lit pas la conversation ; `gh pr merge` et `glab mr merge` restent refusés. | proposition validée |
| Recalage | Avant la fusion, hors de la commande : la pile est rebasée sur la branche par défaut (compétence `stacked-tdd-mr`, « Rebase and conflicts »). La commande refuse une pile en retard, elle ne résout aucun conflit. | proposition validée |
| Mode de fusion | Commit de fusion. Avec lui une pile linéaire et à jour se fusionne sans conflit, et l'arbre final est celui du haut. | choix de conception |
| Forge | GitHub (`gh`). GitLab : même port, adaptateur à écrire dans une MR suivante. | choix de conception |

## Conception

`harness merge-stack [--dry-run] [--validate <commande>] <MR du haut>`, lancé dans le dépôt. La MR se désigne par son numéro, son URL ou sa branche. Sortie standard : une ligne par fait, la dernière est `PASS` ou `FAIL <raison>`. Code de sortie : 0, 1, ou 2 pour un appel mal formé.

1. **Pile.** Depuis la MR désignée, la commande descend de base en base jusqu'à la branche par défaut. Une MR seule est une pile de un.
2. **Contrôles avant toute fusion**, tous nommés d'un coup : MR ouverte et non brouillon ; la branche par défaut est contenue dans la MR du bas (`OUTDATED`) ; chaque branche contient celle du dessous (`NOT STACKED`) ; pipeline de chaque MR terminé et vert (`CHECKS FAIL`). Un seul écart : `FAIL nothing merged`.
3. **Fusion** du bas vers le haut : changement de base vers la branche par défaut, puis fusion, à condition que la tête de la MR soit encore le commit contrôlé. Premier refus : `STOPPED`, `MERGED:`, `REMAINING:`, `FAIL stack partly merged`.
4. **Preuves.** L'arbre de la branche par défaut est celui du haut de la pile (`TREE IDENTICAL`, sinon `TREE DIFFERS` et les fichiers en écart). Puis son pipeline est attendu (`CHECKS PASS`, `CHECKS FAIL`) ; sans pipeline, la commande de `--validate` tourne sur le commit fusionné dans un arbre de travail jetable (`VALIDATION PASS`, `VALIDATION FAIL`) ; sans l'un ni l'autre, `CHECKS NONE` et échec.

Architecture : cas d'usage `app.MergeStack` derrière trois ports (`Forge`, `History`, `Validator`), adaptateurs `adapters/github` (CLI `gh`) et `adapters/git`. Réglages par l'environnement : `HARNESS_POLL_INTERVAL` (15s), `HARNESS_CHECKS_GRACE` (60s, délai après lequel un commit sans pipeline est dit sans pipeline), `HARNESS_CHECKS_TIMEOUT` (30m).

Hook `never-merge` : `harness merge-stack` reçoit `ask`, sauf avec `--dry-run` ou `--help`.

### Non retenu

- Résoudre des conflits dans la commande : un conflit avec un changement extérieur demande un jugement, il se traite au recalage, avant la demande de fusion.
- Fusion par squash ou rebase : chaque MR du dessus entrerait en conflit avec la précédente.
- Retour arrière automatique après un échec.

## Critères d'acceptation

Frontière de tous les critères : la ligne de commande `harness` sur un dépôt git jetable et une forge simulée (`hooks/tests/fake-forge/gh`), sortie standard et code de sortie.

### FUS-001 — Une pile se fusionne du bas vers le haut et la branche par défaut porte le code du haut
- **Étant donné** une pile de trois MR ouvertes `a` ← `b` ← `c`, la première sur `main`, toutes au pipeline vert
- **Quand** `harness merge-stack 3` est lancé dans le dépôt
- **Alors**
  - les trois MR sont fusionnées
  - l'arbre de `main` sur le dépôt distant est celui du haut de la pile avant la fusion
  - la sortie contient `STACK 3: #1 a -> #2 b -> #3 c`, `TREE IDENTICAL` et `CHECKS PASS main`, se termine par `PASS`, et le code de sortie est 0
- **Cas réel** : les quinze MR #55 à #69 fusionnées à la main le 13/11, une par une, avec changement de base et attente du pipeline à chaque fois

### FUS-002 — Une MR seule se fusionne sans celles du dessus
- **Étant donné** la même pile
- **Quand** `harness merge-stack a` est lancé (la MR du bas, désignée par sa branche)
- **Alors**
  - seule la MR du bas est fusionnée, les deux autres restent ouvertes
  - l'arbre de `main` est celui de la branche `a`, et la sortie se termine par `PASS`
- **Cas réel** : demande du 13/11 : « merger une pile ou une MR d'ailleurs »

### FUS-003 — Rien n'est fusionné tant qu'une MR de la pile n'est pas prête
- **Étant donné** la même pile, dont soit la MR du milieu a un pipeline rouge, soit `main` a reçu un commit que la pile ne contient pas
- **Quand** `harness merge-stack 3` est lancé
- **Alors**
  - la sortie nomme l'obstacle (`CHECKS FAIL #2`, ou `OUTDATED #1`), se termine par `FAIL nothing merged`, et le code de sortie est 1
  - aucune MR n'est fusionnée
- **Cas réel** : pile #55 à #69 : #61 et #69 en conflit avec `main` au milieu de la série, découvert après sept fusions

### FUS-004 — La fusion s'arrête à la MR qui résiste et dit ce qui reste
- **Étant donné** la même pile, dont la forge refuse de fusionner la MR du milieu
- **Quand** `harness merge-stack 3` est lancé
- **Alors**
  - la sortie contient `STOPPED #2`, `MERGED: #1`, `REMAINING: #2 #3`, se termine par `FAIL stack partly merged`, et le code de sortie est 1
  - la MR du bas est fusionnée, les deux autres restent ouvertes
- **Cas réel** : demande du 13/11 : « tu arrêtes et tu me notifies »

### FUS-005 — Une branche par défaut qui n'est pas le code du haut de la pile est signalée
- **Étant donné** la même pile, et un tiers qui pousse un commit sur `main` pendant la fusion
- **Quand** `harness merge-stack 3` est lancé
- **Alors** la sortie contient `TREE DIFFERS` avec le nom du fichier en écart, se termine par `FAIL stack merged, main differs from the top of the stack`, et le code de sortie est 1
- **Cas réel** : demande du 13/11 : « ce que je veux à la fin, c'est que le code du haut de la pile (tel qu'il est à la fin) soit présent »

### FUS-006 — Une branche par défaut rouge après la fusion est signalée
- **Étant donné** la même pile, et un pipeline de `main` qui échoue sur le commit fusionné
- **Quand** `harness merge-stack 3` est lancé
- **Alors** la sortie contient `TREE IDENTICAL` puis `CHECKS FAIL main`, se termine par `FAIL stack merged, main is red`, et le code de sortie est 1
- **Cas réel** : demande du 13/11 : « si on merge, on vérifie que la CI ou les tests de validation passent »

### FUS-007 — Sans pipeline, la commande de validation fait foi
- **Étant donné** la même pile, sur un dépôt dont `main` n'a aucun pipeline
- **Quand** `harness merge-stack 3` est lancé avec `--validate '<commande>'`, puis sans
- **Alors**
  - avec une commande qui réussit sur le code fusionné : `VALIDATION PASS`, dernière ligne `PASS`, code de sortie 0, et `git worktree list` est celui d'avant
  - avec une commande qui échoue : `VALIDATION FAIL`, dernière ligne `FAIL stack merged, main is red`, code de sortie 1
  - sans commande : `CHECKS NONE`, dernière ligne `FAIL stack merged, main not verified`, code de sortie 1
- **Cas réel** : règle de `CLAUDE.md` : « If the repository has no pipeline, or it cannot be made green, stop and tell me »

### FUS-008 — Un essai à blanc vérifie la pile sans rien fusionner
- **Étant donné** la même pile
- **Quand** `harness merge-stack --dry-run 3` est lancé
- **Alors**
  - la sortie contient `READY`, se termine par `PASS nothing merged (dry run)`, et le code de sortie est 0
  - aucune MR n'est fusionnée et `main` n'a pas bougé
- **Cas réel** : pile #55 à #69 : l'état de chaque MR a été relu à la main avant de commencer

### FUS-009 — Fusionner une pile passe par l'accord de l'utilisateur
- **Étant donné** le hook `never-merge`
- **Quand** il juge un appel Bash
- **Alors**
  - `harness merge-stack 3` et `~/.local/bin/harness merge-stack --validate "mise run check" c` reçoivent `ask`
  - `harness merge-stack --dry-run 3` passe
  - `gh pr merge 3 --merge` et `glab mr merge 3` reçoivent `deny`
- **Cas réel** : demande du 13/11 : « on a le droit de merge, mais seulement à ma demande »
