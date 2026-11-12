# Recherche de code par intention (Semble) — spec

Date : 2026-11-13. Périmètre : `config.toml`, `tasks/bootstrap`, `claude/agents/semble-search.md`, `claude/CLAUDE.md`, `claude/settings.json`, hook `search-hint` du binaire `harness`.

## Objectif

Demande du 13/11 : « il faudrait qu'on branche "semble" dans notre harness. Pour qu'il fasse la recherche automatiquement. » [Semble](https://github.com/MinishLab/semble) cherche du code par intention (« où est gérée l'authentification ») et rend les extraits utiles, là où grep demande de deviner les mots puis de lire les fichiers.

## Décisions prises (13/11)

| Sujet | Décision | Par |
|---|---|---|
| Redirection de grep vers Semble | Non retenue : grep cherche une chaîne exacte, Semble une intention ; remplacer l'un par l'autre casserait « toutes les occurrences de X ». | proposition validée |
| Rappel ou blocage | Rappel : « Bloquer, ça me semble violent ». La recherche demandée s'exécute toujours. | utilisateur |
| Semble ne trouve rien | Retour à grep, sans arrêt. | utilisateur |
| Installation | Outil du mise global (`pipx:semble`, extra `mcp`) ; serveur MCP déclaré à Claude Code par `bootstrap`. Semble n'est pas un service : l'index se construit à la demande, le serveur MCP est démarré par Claude Code. | choix de conception |

## Conception

- **Règle** (`CLAUDE.md`, « Searching code ») : question sur le code → `semble search` ; chaîne, identifiant ou motif exact → grep ; résultat vide → grep.
- **Agent** `semble-search` (Haiku, outils Bash et Read).
- **Hook `search-hint`** (`PreToolUse`, outils `Bash` et `Grep`) : quand le motif d'une recherche (`Grep`, ou `grep`, `egrep`, `rg`, `ag` dans une ligne Bash, y compris derrière `rtk`) se lit comme une question, il ajoute un rappel au contexte du modèle (`additionalContext`) sans rendre de décision de permission. Un motif se lit comme une question quand il ne porte aucun signe d'expression régulière et qu'il aligne au moins trois mots simples, séparés par des espaces ou par `|`.

### Non retenu

- Compter les grep successifs d'une session : le hook devrait garder un état entre deux appels.
- Indexer au démarrage de la session : la première recherche construit l'index en moins d'une seconde.

## Critères d'acceptation

Frontière : la ligne de commande `harness hook search-hint`, appel d'outil sur l'entrée standard, réponse sur la sortie standard.

### RCH-001 — Une recherche qui se lit comme une question passe avec un rappel de Semble
- **Étant donné** un appel `Grep` de motif `where are tokens refreshed`, un appel Bash `grep -rn "how is the stack merged" harness` et un appel Bash `rtk proxy rg "auth|login|session" src`
- **Quand** le hook `search-hint` juge chacun
- **Alors**
  - la réponse porte un `additionalContext` qui nomme `semble search`
  - elle ne porte aucune `permissionDecision`
- **Cas réel** : cette session, 13/11 : les explorations du dépôt du harnais ont été faites par grep successifs alors que Semble était disponible

### RCH-002 — Une recherche exacte passe en silence
- **Étant donné** un appel `Grep` de motif `TF_PLUGIN_CACHE_DIR`, et les appels Bash `grep -rn "func NewMergeStack" harness`, `rg -A 3 "permissionDecision.*ask" claude` et `git log --oneline -5`
- **Quand** le hook `search-hint` juge chacun
- **Alors** il n'écrit rien et sort à 0
- **Cas réel** : demande du 13/11, « toutes les occurrences de `TF_PLUGIN_CACHE_DIR` » doit rester un grep
