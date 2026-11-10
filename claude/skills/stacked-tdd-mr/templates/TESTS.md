# Tests

## ingress — Exposition HTTP du chart
Branche : `feat/ingress` · Commande : `mise run test:helm`

### ING-002 — Ingress exposé en TLS sur l'hôte configuré
- **Critère** :
  - **Étant donné** `ingress.enabled=true`, `ingress.host=x.example`, `ingress.tls.secretName=x-tls`
  - **Quand** le chart est rendu (`helm template`)
  - **Alors**
    - il existe un Ingress pour l'hôte `x.example`
    - avec TLS via `x-tls`
    - routant vers le Service du chart sur son port nommé `http`
- **Test** : `chart/tests/acceptance/ingress_test.yaml` › `[ING-002] tls on configured host`

<!-- Le critère est recopié en toutes lettres de la spec : c'est le document que l'humain relit, jamais un renvoi vers la spec. Un « Alors » à plusieurs clauses garde sa liste (une clause par élément) ; la porte exige un mutant par clause. -->
