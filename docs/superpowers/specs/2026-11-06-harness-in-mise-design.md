# Harness Claude Code déployé par le dépôt mise

Date : 2026-11-06 · Statut : en revue

## Objectif

Un seul dépôt (`Wihrt/mise`, cloné dans `~/.config/mise`) porte la config mise
et le harness Claude Code. Sur un poste vierge, `mise bootstrap --adopt
Wihrt/mise` suffit à obtenir un `~/.claude` fonctionnel ; toute modification du
harness passe par une PR et sa CI ; l'historique git remplace les fichiers
`*.bak-*`.

## Constat (2026-11-06)

- `~/.claude` n'est pas un dépôt git (`git -C ~/.claude status` : `fatal: not a
  git repository`) ; l'historique tient dans des `*.bak-<date>`.
- `rtk` et `rtk-proxy` sont déjà déclarés dans `config.toml` (1.1.0) ; `bats`
  vient de `~/.claude/hooks/mise.toml` (1.13.0).
- `settings.json` référence les hooks par `~/.claude/hooks/…` : aucun chemin à
  réécrire.
- `~/.claude/skills/synced/` est alimenté par la synchronisation claude.ai ;
  `skills/curl-md` et `skills/find-skills` sont des liens vers `~/.agents/skills`
  (installés par `npx skills`).
- `~/.config/rtk/llm-proxy/` est de l'état (bases SQLite, logs, 71 Mo).
- `pipeline.toml` ne contient aucun secret (`api_key_env` est un nom de
  variable).
- Le plugin `superpowers@anthropic-plugin-directory` est activé ; la marketplace
  `anthropic-plugin-directory` est intégrée à Claude Code.
- `require-worktree.sh` teste le dossier parent du chemin édité sans résoudre
  les liens : un lien *fichier* vers le main checkout échappe au contrôle.
- mise 2026.10.25 : l'étape `dotfiles` de `mise bootstrap` lit les entrées
  `[dotfiles]` ; une `source` relative part du dossier du fichier de config.
  Vérifié en bac à sable : `symlink` crée un lien unique (fichier ou dossier) ;
  `symlink-each` crée de vrais dossiers et ne lie que les fichiers ; un vrai
  fichier à la cible bloque l'application (`refusing to overwrite existing files
  (use --force)`), sans sauvegarde.

## Périmètre

Le dépôt est public (`gh repo view Wihrt/mise` : `PUBLIC`) : le harness versionné ne
contient ni hôte, ni projet, ni chemin propre à un client ou au poste. Les
traces de fabrication des skills (`references/evidence-*.md`, `.ledger.jsonl`,
`.sidecar.json.lock`) restent hors dépôt, copiées dans
`~/.claude/docs/skill-evidence/<skill>/` et ignorées par git. La détection du
forge ne liste aucun hôte : `github.com` → `gh` ; sinon l'hôte d'`origin` connu
de `glab auth status` → `glab`, ou de `gh auth status --hostname <hôte>` →
`gh` ; sinon arrêt.

Versionné : `CLAUDE.md`, `RTK.md`, `settings.json`, `agents/`, `hooks/` (scripts
et tests bats), les 17 skills locaux, et `~/.config/rtk/{config,filters,pipeline}.toml`
+ `pipelines/`.

Hors périmètre : `~/.claude/docs/`, la mémoire (`projects/`), `plugins/`,
`skills/synced/`, `skills/curl-md`, `skills/find-skills`, `skills/.archive/`,
tout `*.bak-*`, `~/.config/rtk/llm-proxy/`.

## Conception

### Arborescence du dépôt

```
claude/
  CLAUDE.md  RTK.md  settings.json
  agents/  hooks/  skills/<nom>/
rtk/
  config.toml  filters.toml  pipeline.toml  pipelines/
```

### Déploiement

Des entrées `[dotfiles]` dans `config.toml`, toutes en `mode = "symlink"` :

- un lien par fichier : `~/.claude/{CLAUDE.md,RTK.md,settings.json}`,
  `~/.config/rtk/{config,filters,pipeline}.toml` ;
- un lien par dossier : `~/.claude/agents`, `~/.claude/hooks`,
  `~/.config/rtk/pipelines` ;
- un lien par skill : `~/.claude/skills/<nom>`. Pas de `symlink-each` sur
  `skills/` : il créerait des dossiers réels où un nouveau fichier sortirait du
  dépôt.

L'étape `dotfiles` de `mise bootstrap` pose les liens ; l'étape `tools`
installe `rtk`, `rtk-proxy` et `bats` ; la tâche `bootstrap` installe ensuite
le plugin superpowers s'il est absent, en appelant la tâche `claude-plugins`
(`claude plugin install superpowers@anthropic-plugin-directory`), qui ne fait
rien si `claude` est absent.

Mise à jour : merge de PR puis `git pull` (ou `mise bootstrap --update`), les
liens pointant déjà vers le checkout.

### Garde-fou d'édition

`require-worktree.sh` résout le chemin édité (`realpath -m`) avant le test du
main checkout. Éditer `~/.claude/CLAUDE.md` revient à éditer le main checkout
et est refusé ; on édite dans un worktree du dépôt.

### Cas `settings.json`

Non vérifié : qu'une écriture de Claude Code dans `settings.json` (`/config`,
permissions) traverse le lien ou le remplace par un fichier. Les deux cas sont
visibles : modification dans `git status` du main checkout, ou cible signalée
par `mise dot status`. À observer après migration.

### Outillage du dépôt et contrôles CI

Fait vérifié (mise 2026.10.25) : dans le dossier de config globale, `mise.toml`
et `mise.local.toml` sont chargés depuis n'importe quel répertoire, et les
fichiers de `tasks/` sont des tâches globales ; `.config/mise.toml` n'est chargé
que dans le dépôt. L'outillage du dépôt (baseline repo-tooling) vit donc dans
`.config/mise.toml`, en tâches inline ; `tasks/` ne contient que des tâches du
poste. `mise run check` (CI et local) enchaîne :

1. `test:hooks` : `bats claude/hooks/tests/` ;
2. `test:dotfiles` : `mise dot apply` dans un `HOME` temporaire puis
   vérification des liens ;
3. `lint:harness` : cohérence skills ↔ entrées `[dotfiles]`, et hygiène (ni
   `*.bak*` ni `/home/arnaud` sous `claude/` et `rtk/`, hors `references/` des
   skills) ;
4. `test:plugins` : la tâche `claude-plugins` avec un `claude` simulé.

### Migration de ce poste (une fois, après merge de la dernière PR)

1. `tar czf ~/.claude-backup-2026-11-06.tgz ~/.claude ~/.config/rtk` (hors
   `projects/`, `plugins/`, `llm-proxy/`) ;
2. `mise dot apply --force` ;
3. `mise dot status` sans écart.

Action sur le `$HOME` de l'utilisateur : lancée seulement sur son feu vert.

### Découpage en PR (pile, un worktree)

0. **Outillage + import** : baseline repo-tooling dans `.config/mise.toml`,
   `claude/` et `rtk/` copiés depuis le poste, cette spec, `test:hooks` en CI.
1. **Hygiène** : `lint:harness` (HAR-006).
2. **Hook `realpath`** dans `claude/hooks/require-worktree.sh` (HAR-002).
3. **Déploiement** : entrées `[dotfiles]`, `test:dotfiles`, cohérence dans
   `lint:harness`, README (HAR-001, HAR-003, HAR-004).
4. **Plugin** : tâche du poste `claude-plugins`, appelée par `bootstrap`
   (HAR-005).

Le hook réel (`~/.claude/hooks`) ne change qu'à la migration : aucun lien
n'existe avant, donc rien à protéger plus tôt.

## Critères d'acceptation

### HAR-001 — Poste vierge : le harness est posé en liens vers le dépôt
- **Étant donné** un `HOME` vide et le dépôt installé comme config globale mise
- **Quand** `mise dot apply` est lancé
- **Alors** chaque cible du périmètre (fichiers, `agents`, `hooks`, chaque skill, fichiers et `pipelines` rtk) existe et se résout (`realpath`) vers le fichier ou dossier correspondant du dépôt, et `HOME/.claude/settings.json` est un JSON valide
- **Frontière** : CLI mise sur un `HOME` temporaire

### HAR-002 — Une édition via un lien vers le main checkout est refusée
- **Étant donné** un dépôt avec un commit, et dans un dossier hors dépôt un lien fichier et un lien dossier vers des fichiers de son main checkout
- **Quand** `require-worktree.sh` reçoit un `Edit` ou un `Write` dont `file_path` passe par l'un de ces liens
- **Alors** il refuse ; il accepte le même chemin passant par un lien vers un worktree lié, et un fichier ordinaire hors dépôt
- **Frontière** : entrée JSON / sortie du hook

### HAR-003 — Les contenus tiers de `~/.claude` et `~/.config/rtk` restent intacts
- **Étant donné** un `HOME` contenant `.claude/skills/synced/x/SKILL.md`, un lien `.claude/skills/curl-md` vers un autre dossier, et `.config/rtk/llm-proxy/rtk-proxy.db`
- **Quand** `mise dot apply` est lancé
- **Alors** ces trois chemins existent toujours, avec le même contenu et la même cible de lien
- **Frontière** : CLI mise sur un `HOME` temporaire

### HAR-004 — Un skill sans entrée de déploiement fait échouer le contrôle
- **Étant donné** le dépôt avec un dossier `claude/skills/<nouveau>/` sans entrée `[dotfiles]`, ou une entrée `[dotfiles]` dont la source n'existe pas
- **Quand** `mise run lint:harness` est lancé
- **Alors** il échoue (code non nul) en nommant le skill ou la source en cause ; sur le dépôt cohérent il réussit
- **Frontière** : tâche mise

### HAR-005 — Le plugin superpowers est installé s'il manque, et seulement alors
- **Étant donné** un `claude` factice sur le `PATH` dont `plugin list` omet (ou contient) `superpowers@anthropic-plugin-directory`
- **Quand** la tâche `claude-plugins` s'exécute
- **Alors** `claude plugin install superpowers@anthropic-plugin-directory` est appelé exactement une fois (ou jamais) ; sans `claude` sur le `PATH`, la tâche réussit sans l'appeler
- **Frontière** : tâche `claude-plugins`, `claude` simulé

### HAR-006 — Rien de personnel ni d'interne versionné
- **Étant donné**, sous `claude/` ou `rtk/`, un fichier `*.bak*`, un fichier contenant `/home/arnaud` hors `claude/skills/*/references/`, ou un fichier contenant un motif listé dans le fichier local non versionné `.config/harness-denylist.local` (un motif par ligne)
- **Quand** `mise run lint:harness` est lancé
- **Alors** il échoue en nommant le fichier (sans afficher le motif) ; sans ces fichiers il réussit ; sans `.config/harness-denylist.local`, seuls les deux premiers contrôles s'appliquent et la tâche le signale sur une ligne
- **Frontière** : tâche mise
