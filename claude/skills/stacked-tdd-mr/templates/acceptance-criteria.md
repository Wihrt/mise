# Critères d'acceptation — gabarit

Chaque spec contient une section `## Critères d'acceptation` (ou `## Acceptance criteria`), de préférence en dernier. Un critère = un titre `### <ID> — <titre>` sous cette section, jusqu'au prochain `## `. L'ID est celui de `TESTS.md` (`ING-002`).

```markdown
## Critères d'acceptation

### ING-002 — Ingress exposé en TLS sur l'hôte configuré
- **Étant donné** `ingress.enabled=true`, `ingress.host=x.example`, `ingress.tls.secretName=x-tls`
- **Quand** le chart est rendu (`helm template`)
- **Alors** il existe un Ingress pour l'hôte `x.example`, avec TLS via `x-tls`, routant vers le Service du chart sur son port nommé `http`
- **Frontière** : rendu Helm
- **Profils** : défaut + `values-prod.yaml`
- **Remplace** : SEC-001
```

`Remplace` est optionnel (mode characterize : IDs des anciens critères que celui-ci remplace) ; `Profils` est optionnel ; un critère universel vaut sur chaque profil.

## Règles
- « Alors » énonce un résultat observable à la frontière. Jamais un helper de template, une fonction, une organisation de fichiers, un nom généré, un ordre, un nombre ou un texte exact, sauf si ce détail EST le contrat (le critère le dit alors).
- Un critère n'existe que s'il protège d'une régression que l'outillage du repo laisserait passer (compilateur et typage, schéma — `values.schema.json`, blocs `validation {}`, pydantic —, `tofu validate`, linters de prek ou `mise run check`). Pas de critère dont l'« Alors » est déjà garanti par l'outillage : Go, type d'un champ ou signature ; Python, type annoté vérifié par mypy ou pydantic ; Helm, valeur refusée par le schéma ; OpenTofu, type de variable ou `validation` simple. Le critère vise le comportement que l'outillage ne voit pas (logique, combinaison de valeurs, relation entre objets). Exception : quand le refus par l'outillage EST le contrat promis, le critère le dit explicitement (« l'outillage garantit X ; ce critère le fige parce que … »).
- **Pas de cas aux limites hypothétiques (règle stricte).** Un critère, et tout test (unitaires compris), ne couvre qu'un cas qui (a) est demandé par l'utilisateur ou écrit dans la spec, ou (b) se présente sur des entrées réelles constatées : un fichier, un dépôt, une donnée qui existe, cité dans le critère (`Cas réel : <chemin ou lien>`). Un cas seulement imaginé (format ancien jamais produit, encodage ou fin de ligne exotique, pagination au-delà du volume réel, entrée malformée qu'aucun producteur n'émet, type de valeur que le code n'atteint jamais) n'a ni critère, ni test, ni code qui le traite. Le doute tranche vers « hors périmètre ». Une remarque de revue ne crée pas de cas réel : elle doit citer l'entrée réelle ou la ligne de spec, sinon elle est écartée.
- Préférer les critères universels (« tout Pod rendu, dans chaque profil, respecte restricted-PSS ») aux exemples isolés.
- Plans : chaque tâche liste les IDs de critères qu'elle livre et la frontière ; aucun code de test d'acceptation dans le plan (l'acceptance-writer l'écrit). Les étapes de tests unitaires sont optionnelles.
- Correctif sans spec (flux Debugging) : l'orchestrateur écrit le critère (ID + Étant donné/Quand/Alors tirés de la cause racine prouvée) dans une section `## Critères d'acceptation` placée À LA FIN du fichier de plan, dans la tâche de correctif ; `--spec` pointe alors vers ce fichier.
