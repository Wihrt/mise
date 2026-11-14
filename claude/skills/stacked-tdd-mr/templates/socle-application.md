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
- **Alors** aucun paquet de `app` n'importe, directement ou par un intermédiaire autre que le domaine, un adapter ni un point d'entrée

### ARC-003 — Seule la racine de composition assemble
- **Étant donné** le code du dépôt
- **Quand** le test d'architecture est lancé
- **Alors** aucun paquet hors du domaine et des cas d'usage n'importe un adapter, sauf la racine de composition et l'adapter lui-même

### Notes pour les preuves
- Les trois critères ne se recouvrent pas : un import direct du domaine vers un adapter ne fait échouer que ARC-001, des cas d'usage vers un adapter que ARC-002, d'un adapter vers un autre que ARC-003. Sans cela la porte refuserait un critère (`REDUNDANT`).
- Mutants types. ARC-001 : le domaine importe une bibliothèque hors liste, ou un adapter. ARC-002 : un cas d'usage importe un adapter. ARC-003 : un adapter en importe un autre.
- En Go, un import du domaine vers un adapter qui importe lui-même le domaine est un cycle que le compilateur refuse : le mutant ne se construit pas. Viser un adapter qui n'importe pas le domaine, ou une bibliothèque tierce.
- Changement neutre type : un paquet de plus dans le domaine, un adapter de plus câblé à la racine de composition.
- Élargir la liste des bibliothèques permises au domaine est un changement de critère (`--ac-changed ARC-001`) : la liste est dans le fichier de test, que l'implementer ne modifie pas.
