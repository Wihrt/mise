# 2. Le code mort se prouve par la couverture des tests de frontière

Date: 2026-11-16

## Status

Accepted

Spec : [`claude/docs/specs/2026-11-16-couverture-de-frontiere-design.md`](../../claude/docs/specs/2026-11-16-couverture-de-frontiere-design.md)

## Context

Le harnais veut trois garanties sur une application Go ou Python : l'architecture hexagonale tenue par des tests, des tests d'acceptation toujours verts, et aucun code mort.

Faits :

- La preuve précédente, le « code minimal », retirait chaque bloc ajouté et relançait la suite une fois par bloc. Mesure du 13/11 sur un dépôt réel : 37 exécutions de la suite pour une tâche, 34 minutes. Elle a été retirée de la porte le 16/11 (`claude/docs/specs/2026-11-13-filets-de-refactoring-design.md`, D1), et la même spec interdit tout contrôle dont la durée grandit avec la quantité de code (lignes 61-64). Rien ne la remplace.
- Les tests d'acceptation n'appellent jamais le code dans leur propre processus : un outil en ligne de commande est lancé en sous-processus (`claude/agents/acceptance-writer.md:49`), une API HTTP ou un front web est appelé sur le conteneur que lance le `compose.yaml` du dépôt (`acceptance-writer.md:52-53`).
- L'image testée est aujourd'hui l'image de production, sans second Dockerfile (`claude/skills/compose-dev/SKILL.md:11`).
- Essais du 16/11 (Go 1.27, Python 3.14, coverage.py 7.17.3) : un service construit avec la mesure, lancé par compose et appelé seulement de l'extérieur, rend sa mesure à l'arrêt, à condition de s'arrêter proprement sur SIGTERM. Un service Go qui ne traite pas SIGTERM sort à 0 %.
- Mesure du 16/11 sur le harnais : ses tests d'acceptation Go exécutent 93,0 % des instructions de `internal/app/gate`. Sur 107 blocs non exécutés hors code testé ailleurs, 39 ne font que remonter une erreur.

Options considérées :

1. Compter tous les tests, unitaires compris, plus un outil statique de code mort. Simple, mais un test unitaire maintient en vie du code qu'aucune entrée réelle n'atteint.
2. Ne compter que les tests de frontière, et mesurer aussi dans le conteneur. Demande une image de test instrumentée.
3. Placer le contrôle dans chaque dépôt applicatif (seuil dans la tâche de test). Aucun code dans le harnais, mais des exceptions que l'auteur du code s'accorde par un commentaire.
4. Placer le contrôle dans la porte, qui lit les rapports écrits par la tâche de test.

## Decision

Nous prouverons l'absence de code mort par la couverture des seuls tests qui passent par la frontière du produit, sur tout le code Go et Python de l'application. La tâche de test du dépôt écrit des rapports de couverture ; la porte les lit dans un mode à part (`harness gate coverage`) et échoue pour tout bloc ou toute ligne jamais exécuté, sauf un bloc Go qui ne fait que remonter une erreur, et sauf les exceptions écrites, avec leur raison, dans un fichier du dépôt que l'auteur du code ne modifie pas. L'image lancée par les tests est construite par le Dockerfile de production avec un argument qui active la mesure ; l'image publiée ne la contient pas.

## Consequences

Plus facile :

- Une ligne n'existe que si un critère en a besoin : la règle d'écriture « rien pour un cas hypothétique » devient vérifiable.
- Le coût est d'une exécution de la suite par jugement, quel que soit le volume de code.
- Go et Python passent par le même jugement, à partir de deux formats de rapport.

Plus difficile :

- L'image testée n'est plus, octet pour octet, l'image publiée : elle en diffère par un argument de construction. C'est une exception écrite à la règle de `compose-dev`.
- Un produit qui ne s'arrête pas proprement ne rend aucune mesure. Le socle l'exige déjà (FAC-004).
- Les tests unitaires ne comptent pas : du code qu'eux seuls atteignent est signalé.
- Un chemin d'erreur Go jamais provoqué est admis sans examen.
- La porte lit du code Go pour reconnaître les blocs de remontée d'erreur.

Ce qui nous ferait revoir la décision :

- une liste d'exceptions qui grossit au lieu de se vider ;
- des écarts de comportement constatés entre l'image instrumentée et l'image publiée ;
- un type de produit dont la mesure ne sort pas (serveur Python à plusieurs processus, non essayé).
