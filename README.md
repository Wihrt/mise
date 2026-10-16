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
| tools          | Installe les outils de `[tools]`                                                                                                 |
| tâche          | [`tasks/bootstrap`](tasks/bootstrap) : clé age et config fnox globale, clé API Anthropic, OpenCommit pour la version Node active |

Chaque étape ne modifie que ce qui diffère de la configuration, et la tâche
peut être relancée sans effet de bord.

## fnox

Au premier passage, la tâche génère `~/.config/fnox/age.txt` (la clé privée du
poste, à ne jamais partager) et `~/.config/fnox/config.toml`, qui chiffre les
secrets pour cette clé. Les secrets d'un poste ne se déchiffrent pas sur un
autre.

## OpenCommit

- **Installation** : le `postinstall` de `node` installe OpenCommit dans chaque
  nouvelle version de Node.js, et `auto_update` installe ces nouvelles versions.
- **Configuration** : `[env]` fixe `OCO_AI_PROVIDER` et `OCO_MODEL`.
- **Clé API** : `OCO_API_KEY` est un secret fnox, demandé au premier
  `mise bootstrap` puis injecté dans chaque shell par `fnox activate`.

La clé s'appelle `OCO_API_KEY` plutôt qu'`ANTHROPIC_API_KEY` : exportée dans
tous les shells, `ANTHROPIC_API_KEY` ferait utiliser à Claude Code la
facturation API au lieu de l'abonnement.

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
