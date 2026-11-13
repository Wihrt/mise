# Porte verte plus rapide — spec

Date : 2026-11-13. Périmètre : le binaire `harness` (`harness/`, mode `gate green`), `claude/skills/stacked-tdd-mr/SKILL.md`.

## Objectif

Demande du 13/11 : rendre `harness gate green --proofs` nettement plus rapide sans changer aucun verdict sur du code.

Mesure du 13/11 (flux-cd, slug `ksw-job`, `--jobs 6`, machine 14 cœurs) : 34 minutes, 50 exécutions de la suite `mise run test:acceptance` (82 tests, 55 à 65 s chacune). Une exécution de référence, 12 preuves de mutants et de neutres jugées 6 de front en quelques minutes, puis 37 exécutions de la preuve de code minimal, l'une après l'autre : l'essentiel de la durée.

Deux causes, lues dans le code au commit `6030846` :

- La preuve de code minimal (`runMinimal`, `harness/internal/app/gate/proofs.go:283-320`) retire un bloc dans l'arbre de travail, lance toute la tâche, restaure l'arbre, puis passe au bloc suivant. `--jobs` ne s'y applique pas : il ne parallélise que les mutants et les neutres (`judgeAll`, `proofs.go:148-183`).
- 7 des 37 blocs étaient de la documentation ou des commentaires (README, docstring, commentaires YAML) : forcément `UNNEEDED`, jugés quand même, puis repassés en `--unneeded-ok` à chaque relance.

`characterize` ne lance pas la preuve de code minimal (`characterize.go:77`) : cette spec ne le change pas.

## Décisions prises (13/11)

| Sujet | Décision | Par |
|---|---|---|
| Périmètre | La preuve de code minimal en parallèle, et l'exclusion de la documentation et des commentaires. Hors de cette pile : `--focus-task`, l'arrêt propre sur SIGINT/SIGTERM, `--unneeded-ok` par recouvrement. | utilisateur |
| Documentation | « On fait des tests sur de la doc, donc pas la peine. » Une liste de motifs intégrée à la porte : un bloc ajouté à un fichier de documentation n'est jamais jugé, sans option à passer. `--minimal-exclude <motif>` allonge la liste. | utilisateur |
| Commentaires | « Les commentaires ne sont pas du code, mais une explication » ; pas de mécanisme fin. Règle simple : un bloc dont chaque ligne est vide ou commence par `#` ou `//` n'est pas jugé. | utilisateur |
| Parallélisme | Le mécanisme des mutants (`judgeInWorktree`) : un worktree jetable de `HEAD` par bloc, sous `--jobs`. | brief du 13/11 |
| Liste intégrée | `*.md`, `*.rst`, `*.adoc`. `*.txt`, proposé d'abord, est retiré : il couvrirait `requirements.txt`, qui est du code. | choix de conception |
| Trace | Chaque bloc sauté imprime `MINIMAL EXCLUDED <fichier>:<début>-<fin>` : rien ne disparaît du journal en silence. | proposition validée |
| Délai dépassé | Inchangé : dans la preuve de code minimal, une tâche qui dépasse son délai compte comme un échec, donc le bloc est dit nécessaire (`proofs.go:309`). Limite connue : sous `--jobs`, une suite ralentie par la charge peut ainsi taire un `UNNEEDED`. | choix de conception |
| Mesure | Le gain se prouve par une durée mesurée sur un dépôt d'exemple dont la suite dort (brief du 13/11). Le critère compare deux durées de la même porte, il ne fixe aucun seuil absolu. | utilisateur, forme : choix de conception |
| Identifiants | Les critères continuent la série `PRV` (preuves de la porte), à partir de `PRV-020`. | choix de conception |

## Conception

### Preuve de code minimal en parallèle

Avec `--jobs N` (N > 1), `green --proofs` juge les blocs N à la fois. Pour chaque bloc : un worktree jetable de `HEAD` (`git worktree add --detach`, sous le verrou `Gate.worktrees`), le bloc retiré dans ce worktree, la tâche lancée dans ce worktree, puis le worktree retiré. L'arbre de travail du dépôt n'est pas touché.

Le verdict d'un bloc ne change pas : le bloc est nécessaire quand la tâche échoue ou n'écrit pas un rapport exploitable, sinon `UNNEEDED`, ou `UNNEEDED ACCEPTED` avec `--unneeded-ok`. Le journal de chaque exécution est retenu puis imprimé quand vient le tour du bloc : les lignes sortent dans l'ordre des blocs, comme sans `--jobs`.

Sans `--jobs`, ou avec `--jobs 1`, le chemin actuel ne change pas (retrait dans l'arbre de travail, puis restauration).

Conditions d'emploi : celles de `--jobs` pour les mutants (`stacked-tdd-mr`, étape 4b). La tâche de test doit tourner depuis une copie fraîche du dépôt et n'utiliser ni port fixe ni chemin partagé.

### Blocs qui ne sont pas du code

Avant de retirer un bloc, la porte l'écarte quand l'une de ces règles s'applique, avec ou sans `--jobs` :

1. **Fichier de documentation.** Le chemin du fichier correspond à un motif de la liste intégrée (`*.md`, `*.rst`, `*.adoc`) ou à un motif donné par `--minimal-exclude <motif>` (répétable). Les motifs sont ceux de `tdd.Fnmatch` : `*` traverse les `/`.
2. **Bloc de commentaires.** Chaque ligne du bloc, espaces de tête retirés, est vide ou commence par `#` ou `//`.

Un bloc écarté imprime `MINIMAL EXCLUDED <fichier>:<début>-<fin>` à la place qu'aurait eue sa ligne `UNNEEDED`, et la tâche n'est pas lancée pour lui. Le fichier de `--spec` reste sauté sans aucune ligne, comme aujourd'hui (PRV-014).

Limites connues, acceptées : les docstrings et les commentaires `/* */` restent jugés et passent par `--unneeded-ok` ; un bloc fait seulement de lignes à effet qui commencent par `#` ou `//` (`#!/bin/sh`, `//go:build`) passe sans preuve.

### Ce qui change pour un même dépôt

- Sans `--jobs` et sans bloc de documentation ni de commentaires : sortie identique à aujourd'hui.
- Avec `--jobs` : mêmes lignes de verdict, dans le même ordre, en moins de temps.
- Un bloc de documentation ou de commentaires : `MINIMAL EXCLUDED` au lieu de `UNNEEDED` ou de rien. C'est le seul changement de verdict, voulu.

### Documentation de la compétence

`stacked-tdd-mr/SKILL.md` : l'étape 8 lance la preuve de code minimal avec `--jobs` ; `MINIMAL EXCLUDED` rejoint la liste des lignes de la porte ; `--unneeded-ok` ne cite plus le commentaire comme cas d'emploi.

### Approches

- **Parallélisme, retenu** : worktrees jetables sous `--jobs`, chemin actuel sans `--jobs`. C'est le mécanisme déjà prouvé pour les mutants (PRV-019).
- **Non retenu** : toujours passer par des worktrees, même sans `--jobs` (change le comportement par défaut, alors que les mutants resteraient jugés en place) ; transformer chaque bloc en patch pour le circuit des mutants (les contrôles de patch de mutant n'ont pas de sens pour un bloc).
- **Documentation, retenu** : liste intégrée plus option. **Non retenu** : option seule (à passer à chaque lancement, oubliable) ; liste déclarée dans le dépôt (une configuration de plus à concevoir).
- **Commentaires, retenu** : règle sur le début des lignes. **Non retenu** : ne rien faire (une exécution de la suite et une option par bloc) ; reconnaître les commentaires par langage (mécanisme fin, écarté par l'utilisateur).

## Critères d'acceptation

Frontière de tous les critères : la ligne de commande `harness gate green --proofs` sur un dépôt git jetable, sortie standard, code de sortie et durée de la commande.

### PRV-020 — Le code minimal se juge plus vite avec `--jobs`
- **Étant donné** un commit d'acceptation, un commit `feat` qui ajoute quatre blocs séparés dans un fichier de production, et une tâche de test dont chaque exécution dure un temps fixe
- **Quand** `harness gate green --proofs` est lancé sans `--jobs`, puis avec `--jobs 4`
- **Alors** la porte lancée avec `--jobs 4` se termine plus tôt que la porte lancée sans `--jobs`, d'au moins deux fois la durée d'une exécution de la tâche
- **Cas réel** : porte du 13/11 sur flux-cd, slug `ksw-job`, `--jobs 6` : 37 exécutions de la preuve de code minimal l'une après l'autre, 34 minutes

### PRV-021 — Avec `--jobs`, le code minimal rend les mêmes verdicts et laisse le dépôt intact
- **Étant donné** un commit d'acceptation et un commit `feat` qui ajoute, dans un fichier de production, deux blocs séparés : l'un sans lequel le test du critère échoue, l'autre dont le retrait laisse tous les tests verts
- **Quand** `harness gate green --proofs` est lancé avec `--jobs 2`
- **Alors**
  - la sortie contient `UNNEEDED` avec le fichier et les lignes du second bloc, aucune ligne `UNNEEDED` pour le premier, et se termine par `FAIL`
  - relancé avec `--unneeded-ok` pour ce bloc, elle contient `UNNEEDED ACCEPTED` avec le même fichier et les mêmes lignes, aucune autre ligne `UNNEEDED`, et se termine par `PASS`
  - pendant l'exécution, aucune exécution de la tâche de test ne tourne dans le dépôt lui-même avec un bloc retiré : chaque bloc est retiré dans une copie du dépôt
  - après l'exécution, `git worktree list` et `git status --short` du dépôt sont ceux d'avant
- **Cas réel** : `scripts/killswitch.py` laissé amputé de 2 lignes le 13/11, après l'arrêt d'une porte qui retirait les blocs dans l'arbre de travail

### PRV-022 — Un bloc de documentation n'est pas jugé
- **Étant donné** un commit d'acceptation et un commit `feat` qui ajoute un bloc à un fichier de documentation (`README.md` ; de même un fichier `.rst` et un fichier `.adoc`) et, dans un fichier de production, un bloc sans lequel le test du critère échoue
- **Quand** `harness gate green --proofs` est lancé
- **Alors**
  - la sortie contient `MINIMAL EXCLUDED` avec le fichier et les lignes du bloc de documentation, aucune ligne `UNNEEDED`, et se termine par `PASS`
  - la tâche de test est lancée autant de fois que pour le même commit sans le bloc de documentation
- **Cas réel** : porte du 13/11 sur flux-cd, slug `ksw-job` : un bloc de README jugé `UNNEEDED`, repassé en `--unneeded-ok` à chaque relance

### PRV-023 — `--minimal-exclude` écarte d'autres fichiers
- **Étant donné** un commit d'acceptation et un commit `feat` qui ajoute, dans un fichier que la liste intégrée ne couvre pas (`NOTICE`), un bloc dont le retrait laisse tous les tests verts
- **Quand** `harness gate green --proofs` est lancé avec `--minimal-exclude NOTICE`, puis sans
- **Alors**
  - avec l'option, la sortie contient `MINIMAL EXCLUDED` avec le fichier et les lignes du bloc, aucune ligne `UNNEEDED`, et se termine par `PASS`
  - sans l'option, elle contient `UNNEEDED` avec le fichier et les lignes du bloc et se termine par `FAIL`
- **Cas réel** : brief du 13/11, piste 3 : « option ou configuration (`--minimal-exclude '*.md'`) »

### PRV-024 — Un bloc de commentaires n'est pas jugé
- **Étant donné** un commit d'acceptation et un commit `feat` qui ajoute, dans des fichiers de production : un bloc dont chaque ligne est vide ou commence par `#` ; un bloc dont chaque ligne commence par `//` ; un bloc qui mêle une ligne de commentaire et une ligne de code, et dont le retrait laisse tous les tests verts
- **Quand** `harness gate green --proofs` est lancé
- **Alors**
  - la sortie contient `MINIMAL EXCLUDED` avec le fichier et les lignes de chacun des deux blocs de commentaires, et aucune ligne `UNNEEDED` pour eux
  - elle contient `UNNEEDED` avec le fichier et les lignes du bloc mêlé, et se termine par `FAIL`
- **Cas réel** : porte du 13/11 sur flux-cd, slug `ksw-job` : des commentaires YAML jugés `UNNEEDED`, repassés en `--unneeded-ok` à chaque relance
