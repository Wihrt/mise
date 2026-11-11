# Critères d'acceptation — gabarit

Chaque spec contient une section `## Critères d'acceptation` (ou `## Acceptance criteria`), de préférence en dernier. Un critère = un titre `### <ID> — <titre>` sous cette section, jusqu'au prochain `## `. L'ID est celui de `TESTS.md` (`ING-002`).

```markdown
## Critères d'acceptation

### ING-002 — Ingress exposé en TLS sur l'hôte configuré
- **Étant donné** `ingress.enabled=true`, `ingress.host=x.example`, `ingress.tls.secretName=x-tls`
- **Quand** le chart est rendu (`helm template`)
- **Alors**
  - il existe un Ingress pour l'hôte `x.example`
  - cet Ingress porte TLS via `x-tls`
  - il route vers le Service du chart sur son port nommé `http`
- **Frontière** : rendu Helm
- **Profils** : défaut + `values-prod.yaml`
- **Remplace** : SEC-001
```

**Forme du « Alors ».** Une ligne = une clause ; une ligne suivie d'une liste = une clause par élément. La porte compte les clauses et exige un mutant par clause : écrire chaque promesse indépendante sur sa propre ligne, pas deux promesses dans une phrase. Cas réel : un critère à trois promesses sur une ligne n'était prouvé que pour l'une d'elles (état des lieux, leçon 9).

`Remplace` est optionnel (mode characterize : IDs des anciens critères que celui-ci remplace) ; `Profils` est optionnel ; un critère universel vaut sur chaque profil.

## Règles
- « Alors » énonce un résultat observable à la frontière. Jamais un helper de template, une fonction, une organisation de fichiers, un nom généré, un ordre, un nombre ou un texte exact, sauf si ce détail EST le contrat (le critère le dit alors).
- Un critère n'existe que s'il protège d'une régression que l'outillage du repo laisserait passer (compilateur et typage, schéma — `values.schema.json`, blocs `validation {}`, pydantic —, `tofu validate`, linters de prek ou `mise run check`). Pas de critère dont l'« Alors » est déjà garanti par l'outillage : Go, type d'un champ ou signature ; Python, type annoté vérifié par mypy ou pydantic ; Helm, valeur refusée par le schéma ; OpenTofu, type de variable ou `validation` simple. Le critère vise le comportement que l'outillage ne voit pas (logique, combinaison de valeurs, relation entre objets). Exception : quand le refus par l'outillage EST le contrat promis, le critère le dit explicitement (« l'outillage garantit X ; ce critère le fige parce que … »).
- **Pas de cas aux limites hypothétiques (règle stricte).** Un critère, et tout test (unitaires compris), ne couvre qu'un cas qui (a) est demandé par l'utilisateur ou écrit dans la spec, ou (b) se présente sur des entrées réelles constatées : un fichier, un dépôt, une donnée qui existe, cité dans le critère (`Cas réel : <chemin ou lien>`). Un cas seulement imaginé (format ancien jamais produit, encodage ou fin de ligne exotique, pagination au-delà du volume réel, entrée malformée qu'aucun producteur n'émet, type de valeur que le code n'atteint jamais) n'a ni critère, ni test, ni code qui le traite. Le doute tranche vers « hors périmètre ». Une remarque de revue ne crée pas de cas réel : elle doit citer l'entrée réelle ou la ligne de spec, sinon elle est écartée.
- Préférer les critères universels (« tout Pod rendu, dans chaque profil, respecte restricted-PSS ») aux exemples isolés.
- Plans : chaque tâche liste les IDs de critères qu'elle livre et la frontière ; aucun code de test d'acceptation dans le plan (l'acceptance-writer l'écrit). Les étapes de tests unitaires sont optionnelles.
- Correctif sans spec (flux Debugging) : l'orchestrateur écrit le critère (ID + Étant donné/Quand/Alors tirés de la cause racine prouvée) dans une section `## Critères d'acceptation` placée À LA FIN du fichier de plan, dans la tâche de correctif ; `--spec` pointe alors vers ce fichier.

## Grille unique : garder ou refuser un critère
Quatre raisons de garder (au moins une) :
1. Relation entre deux endroits du code : un test de relation fait varier un côté et observe l'autre dans le même test.
2. Logique : branche, calcul, combinaison de valeurs que l'outillage ne voit pas.
3. Contrat avec l'extérieur : ce qu'un consommateur du livrable lit ou appelle.
4. Incident réel : une régression déjà vue (citer `Cas réel :`).

Six raisons de refuser, chacune avec la preuve de la porte qui la détecte :

| Raison de refuser | Preuve qui la détecte |
|---|---|
| Teste l'outil (compilateur, schéma, linter) plutôt que le livrable | critère refusé à l'écriture ; sinon aucun mutant de production ne le fait rougir (`MUTANT MISSING`, `SURVIVED`) |
| Recopie la configuration (le « Alors » répète une valeur du code) | `NO-NEUTRAL <ID>` : aucun changement neutre n'existe |
| Fige un détail (nom généré, ordre, nombre, texte exact) | `BRITTLE <ID>` : un changement neutre fait échouer le test |
| Cas hypothétique | aucune : règle de lecture, `Cas réel :` exigé |
| Doublon d'un autre critère | `REDUNDANT <ID>: covered by <ID>` |
| Ne peut pas échouer | `SURVIVED <ID>` / `MUTANT MISSING` : le mutant ne fait rougir aucun test |

## Catalogue
Un ensemble qui a vocation à grandir (liste d'environnements, d'applications, de releases) n'a ni nombre ni noms figés dans un critère : le critère porte sur la règle qui s'applique à chaque élément, et ajouter ou retirer un élément ne change aucun test. Preuve : le changement neutre « élément ajouté à l'ensemble » (`BRITTLE`).

## Règle de plan
- Une tâche = un fichier de test : les tâches d'un plan se découpent par fichier de test, car un fichier cité par une section antérieure ne peut changer qu'avec `--replaces` ou `--ac-changed` (`KEPT TEST MODIFIED`).
- La tâche de test écrit un rapport par test à un chemin qui lui est propre et n'a aucun fichier intermédiaire partagé avec une autre tâche de test : `check` lance ses dépendances en parallèle.
- La tâche de test de la `Commande` d'une section est lancée par un hook de commit ou un job CI du dépôt, directement ou par une tâche qui en dépend (`NOT WIRED <tâche>`) ; la brancher avant la première acceptation.
