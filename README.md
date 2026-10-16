# Configuration mise globale

Ce dépôt est la configuration mise globale d'un poste (`~/.config/mise`).
Il installe les outils, active mise et fnox dans bash, amorce fnox et
installe OpenCommit.

## Déployer un poste

```sh
curl -fsSL https://mise.run | sh
export PATH="$HOME/.local/bin:$PATH"
mise bootstrap --adopt Wihrt/mise
```

Si `git` manque, mise propose de l'installer avec le gestionnaire de paquets
avant de cloner.

Au premier passage, mise installe les outils avant que la tâche `bootstrap`
ne demande le `GITHUB_TOKEN`. Beaucoup d'outils viennent de GitHub, dont l'API
limite les appels anonymes à 60 par heure : passez le token dès l'adoption.

```sh
read -rsp "Token GitHub : " GITHUB_TOKEN && export GITHUB_TOKEN
mise bootstrap --adopt Wihrt/mise
```

Si l'installation des outils échoue malgré tout, relancez `mise bootstrap` :
il reprend là où il s'est arrêté.

`--adopt` clone ce dépôt dans `~/.config/mise`, qui doit être absent ou vide,
puis lance `mise bootstrap`. Sur un poste existant, déplacez d'abord l'ancien
`~/.config/mise`. Ouvrez ensuite un nouveau shell.

Pour mettre un poste à jour après un changement du dépôt :

```sh
mise bootstrap --update
```

## Ce que fait `mise bootstrap`

| Étape          | Effet                                                                                                                            |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| dotfiles       | Ajoute `~/.local/bin` au `PATH` de `~/.bashrc` et `~/.bash_profile`, et le bloc `fnox activate` dans `~/.bashrc`                 |
| shell          | Ajoute l'activation de mise : complète dans `~/.bashrc`, par shims dans `~/.bash_profile`                                        |
| packages       | Installe Docker (`docker.io`, Compose, Buildx) et `skopeo` avec apt (sudo)                                                                         |
| tools          | Installe les outils de `[tools]`                                                                                                 |
| tâche          | [`tasks/bootstrap`](tasks/bootstrap) : clé age et config fnox globale, `GITHUB_TOKEN` et clé API Anthropic, OpenCommit pour la version Node active, identité et authentification git GitHub, `rtk init --global`, groupe `docker`, complétions bash |

Chaque étape ne modifie que ce qui diffère de la configuration, et la tâche
peut être relancée sans effet de bord.

## fnox

Au premier passage, la tâche génère `~/.config/fnox/age.txt` (la clé privée du
poste, à ne jamais partager) et `~/.config/fnox/config.toml`, qui chiffre les
secrets pour cette clé. Les secrets d'un poste ne se déchiffrent pas sur un
autre.

La tâche demande ensuite, une seule fois par poste, les secrets globaux que
`fnox activate` exporte dans chaque shell :

| Secret         | Usage                                                         |
| -------------- | ------------------------------------------------------------- |
| `GITHUB_TOKEN` | `gh`, et mise pour éviter la limite de l'API GitHub           |
| `OCO_API_KEY`  | Clé API Anthropic d'OpenCommit                                |

Pour changer une valeur : `fnox set --global <SECRET>`.

## OpenCommit

- **Installation** : le `postinstall` de `node` installe OpenCommit dans chaque
  nouvelle version de Node.js, et `auto_update` installe ces nouvelles versions.
- **Configuration** : `[env]` fixe `OCO_AI_PROVIDER` et `OCO_MODEL`.
- **Clé API** : `OCO_API_KEY` est un secret fnox, demandé au premier
  `mise bootstrap` puis injecté dans chaque shell par `fnox activate`.

La clé s'appelle `OCO_API_KEY` plutôt qu'`ANTHROPIC_API_KEY` : exportée dans
tous les shells, `ANTHROPIC_API_KEY` ferait utiliser à Claude Code la
facturation API au lieu de l'abonnement.

### Hook git

La tâche `bootstrap` installe le hook `prepare-commit-msg` d'OpenCommit dans
`~/.config/git/template/hooks` et règle `init.templateDir` : chaque dépôt créé
par `git init` ou `git clone` le reçoit. Pour un dépôt existant, lancez
`git init` dedans (git n'écrase aucun hook déjà présent) ou `oco hook set`.

Le hook ne génère un message que pour un `git commit` sans message : `-m`, un
merge ou un amend ne l'appellent pas. Si l'appel échoue, l'erreur apparaît en
commentaire dans le message et le commit n'est pas bloqué.

`core.hooksPath` n'est pas utilisé : il désactiverait les hooks propres à
chaque dépôt, comme ceux de prek.

Pour changer la clé :

```sh
fnox set --global OCO_API_KEY
```

## rtk

Le `postinstall` de `rtk` lance `rtk init --global --auto-patch` à chaque
nouvelle version : hook Claude Code dans `~/.claude/settings.json` et
`~/.claude/RTK.md`, sans question. La tâche `bootstrap` le relance aussi pour
une version déjà installée ; la commande est idempotente.

## Docker

Docker Engine vient d'apt : `docker.io`, `docker-compose-v2` pour
`docker compose`, et `docker-buildx`. Le paquet active `dockerd` à
l'installation. La tâche `bootstrap` ajoute l'utilisateur au groupe `docker`
(sudo) : c'est effectif à la prochaine connexion, ou tout de suite avec
`newgrp docker`. kind et tilt utilisent Docker sans réglage supplémentaire.

`skopeo` (apt) copie et inspecte des images d'un registre à l'autre sans démon.

## Git

La tâche `bootstrap` configure git globalement :

- l'identité (`user.name`, `user.email`), reprise du compte GitHub du
  `GITHUB_TOKEN` si elle n'est pas déjà définie. Un e-mail GitHub privé est
  remplacé par l'adresse `noreply` du compte ;
- l'authentification HTTPS sur GitHub par `gh`, via son shim mise.
  `gh auth setup-git` écrirait le chemin versionné de `gh`, qui casse à chaque
  mise à jour.

L'authentification sur les instances GitLab reste à configurer à la main.

## Complétions bash

La tâche génère les complétions des outils qui en fournissent dans
`~/.local/share/bash-completion/completions`. bash-completion les charge à la
première touche Tab, sans ralentir le démarrage du shell. Elles sont
régénérées à chaque `mise bootstrap`.
