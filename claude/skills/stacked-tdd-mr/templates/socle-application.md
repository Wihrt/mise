# Socle d'une application Go ou Python

Critères que toute application porte. À recopier dans la spec du dépôt ; l'acceptance-writer les transcrit dans `TESTS.md`. Les IDs sont réservés : aucun autre critère ne s'appelle `ARC-…` ni `FAC-…`. Compléter chaque critère par son `Cas réel` dans le dépôt.

## architecture — Les couches hexagonales ne se mélangent pas

Tout dépôt Go ou Python. Un test par critère, sous `tests/architecture/` (gabarits : `hexagonal-go/templates/arch_test.go`, `hexagonal-py/templates/test_architecture.py`).

### ARC-001 — Le domaine ne dépend que de lui-même
- **Étant donné** le code du dépôt
- **Quand** le test d'architecture est lancé
- **Alors** aucun paquet du domaine n'importe, directement ou par un intermédiaire, autre chose que le domaine, la bibliothèque standard et les bibliothèques que le fichier de test énumère

### ARC-002 — Les cas d'usage ne connaissent aucune technologie
- **Étant donné** le code du dépôt
- **Quand** le test d'architecture est lancé
- **Alors** aucun paquet de `app` n'importe, directement ou par un intermédiaire autre que le domaine, autre chose que les cas d'usage, le domaine, la bibliothèque standard et les bibliothèques que le fichier de test énumère pour eux

### ARC-003 — Seule la racine de composition assemble
- **Étant donné** le code du dépôt
- **Quand** le test d'architecture est lancé
- **Alors** aucun paquet hors du domaine et des cas d'usage n'importe un adapter, sauf la racine de composition et l'adapter lui-même

### Notes pour les preuves
- Les trois critères ne se recouvrent pas : un import direct du domaine vers un adapter ne fait échouer que ARC-001, des cas d'usage vers un adapter que ARC-002, d'un adapter vers un autre que ARC-003. Sans cela la porte refuserait un critère (`REDUNDANT`).
- Mutants types. ARC-001 : le domaine importe une bibliothèque hors liste, ou un adapter. ARC-002 : un cas d'usage importe un adapter, ou une bibliothèque hors liste (un ORM, un client HTTP). ARC-003 : un adapter en importe un autre.
- En Go, un import du domaine vers un adapter qui importe lui-même le domaine est un cycle que le compilateur refuse : le mutant ne se construit pas. Viser un adapter qui n'importe pas le domaine, ou une bibliothèque tierce.
- Changement neutre type : un paquet de plus dans le domaine, un adapter de plus câblé à la racine de composition.
- Élargir une liste de bibliothèques permises est un changement de critère (`--ac-changed ARC-001` pour celle du domaine, `--ac-changed ARC-002` pour celle des cas d'usage) : les deux listes sont dans le fichier de test, que l'implementer ne modifie pas.

## douze-facteurs — L'image se comporte en application à douze facteurs

Dépôt Go ou Python avec un `Dockerfile` à la racine. Le produit est l'image, lancée par le `compose.yaml` du dépôt (`compose-dev`). Tests natifs du langage sous `tests/container/`, tâche `test:container`. Détail de chaque facteur : skill `twelve-factor`.

### FAC-001 — Le port d'écoute vient de l'environnement
- **Étant donné** l'application lancée par le compose avec un port d'écoute autre que celui par défaut
- **Quand** une requête est envoyée sur ce port
- **Alors** l'application répond

### FAC-002 — Une variable obligatoire absente arrête le démarrage
- **Étant donné** l'image construite par le compose, lancée sans une variable d'environnement obligatoire
- **Quand** le conteneur démarre
- **Alors**
  - il sort avec un code non nul
  - sa sortie nomme la variable manquante

### FAC-003 — Les logs sortent sur la sortie standard
- **Étant donné** l'application lancée par le compose
- **Quand** une requête est servie
- **Alors** la sortie du conteneur porte la ligne de log de cette requête

### FAC-004 — L'arrêt est propre
- **Étant donné** l'application lancée par le compose
- **Quand** le conteneur reçoit SIGTERM
- **Alors** il sort avec le code 0 avant la fin du délai d'arrêt

### FAC-005 — Aucun état local
- **Étant donné** l'application lancée par le compose
- **Quand** elle sert son cas nominal
- **Alors**
  - le système de fichiers racine du conteneur est en lecture seule
  - le processus ne tourne pas en root

### Notes pour les preuves
- Non vérifié : ces critères ont été écrits le 14/11 sur un poste sans Docker. Le premier dépôt qui les porte les prouve, et corrige ce gabarit si un critère ne se laisse pas tester ainsi.
- Un critère sans objet (FAC-001 pour un worker sans port) s'écarte par un `Ruling:` consigné.
- Mutants types. FAC-001 : port écrit en dur. FAC-002 : valeur par défaut donnée à la variable obligatoire. FAC-003 : log de requête retiré, ou écrit dans un fichier. FAC-004 : le traitement de SIGTERM retiré du code (Go : `signal.NotifyContext` ; Python : la boucle du worker ignore le signal) : code non nul, ou 137 à la fin du délai. Pas l'`ENTRYPOINT` en forme shell : une image distroless n'a pas de shell, le conteneur ne démarre pas et tous les critères échouent. FAC-005 : `read_only: true` retiré du compose ; `USER` retiré du `Dockerfile`.
- Changement neutre type : une route de plus, un autre port, un autre format de log.
- Comment lancer FAC-001 (un autre `PORT`) et FAC-002 (l'image seule, sans la variable) : `compose-dev`, « Started by the tests ».
