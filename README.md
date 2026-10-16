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
| packages       | Installe `podman`, `buildah` et `skopeo` avec apt (mot de passe sudo)                                                                         |
| tools          | Installe les outils de `[tools]`                                                                                                 |
| tâche          | [`tasks/bootstrap`](tasks/bootstrap) : clé age et config fnox globale, `GITHUB_TOKEN` et clé API Anthropic, OpenCommit pour la version Node active, `rtk init --global`, socket `podman.socket` |

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

## Réglages propres à un poste

Mettez-les dans `~/.config/mise/config.local.toml`, que git ignore. Par exemple,
un proxy :

```toml
[env]
HTTPS_PROXY = "http://proxy:80"
```

## rtk

Le `postinstall` de `rtk` lance `rtk init --global --auto-patch` à chaque
nouvelle version : hook Claude Code dans `~/.claude/settings.json` et
`~/.claude/RTK.md`, sans question. La tâche `bootstrap` le relance aussi pour
une version déjà installée ; la commande est idempotente.

## Podman

`podman` vient d'apt : le podman publié sur GitHub, que mise saurait installer,
n'est qu'un client distant. Podman n'a pas besoin de Docker. `buildah` et
`skopeo` viennent aussi d'apt : ils ne sont pas dans le registre mise, et le
paquet `podman` ne fait que recommander `buildah`.

- `[shell_alias]` définit `docker = "podman"` dans chaque shell où mise est
  activé. Comme tout alias, il n'existe pas dans les scripts, les tâches mise
  ni `mise exec`.
- La tâche `bootstrap` active le socket utilisateur `podman.socket`, et
  `DOCKER_HOST` le désigne : lazydocker et les SDK Docker parlent à Podman.
