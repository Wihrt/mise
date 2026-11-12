# Harnais de test déterministe — spec

Date : 2026-11-10. Périmètre : `~/.claude/agents/bin/tdd-gate` et ses tests, `~/.claude/skills/stacked-tdd-mr/` (compétence et gabarits), `~/.claude/agents/{acceptance-writer,mutant-writer,implementer}.md`, deux lignes de `~/.claude/skills/repo-tooling/`.

État des lieux et cas réels : `.superpowers/sdd/2026-11-10-elagage-tests/harnais/etat-des-lieux.md` et https://claude.ai/artifact/P1gsNTBiE2eStS5PyVo8TD.

## Objectif

Demande du 10/11 : « que ça génère les tests qui sont strictement nécessaires et qui protègent vraiment d'une régression. De cette manière, le code ne régresse pas entre deux fonctionnalités. »

Aujourd'hui, la porte prouve une seule chose : un test peut échouer. Tout le reste (ce test est-il utile, recopie-t-il la configuration, fait-il doublon) repose sur le jugement d'un agent ou d'un relecteur. La revue des `TESTS.md` a montré le résultat : sur 441 entrées jugées à retirer dans les huit inventaires `scope.psv` du chantier, 164 recopiaient la configuration, 100 faisaient doublon, 18 figeaient un détail, 4 ne pouvaient pas échouer.

**Déterministe** veut dire ici : un test n'entre dans `TESTS.md`, et n'y reste, que si la porte a exécuté trois preuves. Le verdict ne dépend plus de qui relit.

| Preuve | Question | Ce que la porte exécute | Refus |
|---|---|---|---|
| **Sensible** | Le test échoue-t-il quand le comportement promis casse ? | Un mutant du code de production par clause du « Alors » | `SURVIVED`, `MUTANT MISSING` |
| **Spécifique** | Le test laisse-t-il passer un changement légitime ? | Un changement neutre par critère : une modification du dépôt après laquelle le « Alors » reste vrai | `BRITTLE`, `NEUTRAL MISSING` |
| **Nécessaire** | Le test apporte-t-il quelque chose qu'un autre n'apporte pas ? | La table « quel mutant tue quels critères » | `REDUNDANT` |

Et une garantie dans le temps : ce qui a été prouvé ne s'affaiblit pas à la fonctionnalité suivante (test gardé inchangé, rejeu de mutants quand des tests sont remplacés).

Ce que les preuves n'attrapent pas : le test de l'outil (36 entrées) et le cas hypothétique (119). Ils restent des règles d'écriture du critère, lues par l'auteur de la spec et tranchées à la relecture de la spec par l'utilisateur.

## Décisions prises (10/11)

- H1 : les trois lots (textes, contrôles de la porte, preuve par mutants étendue).
- H2 : le contrôle « critère en toutes lettres » ne s'applique qu'au slug en cours d'écriture ; une tâche de réécriture des tests est prévue (voir « Réécriture »).
- H3 : la porte accepte un mutant dans `mise.toml` hors tâches de test.
- H4 : `~/.claude` reste hors git pour ce lot ; chaque fichier modifié garde une copie `.bak-2026-11-10` ; les tests de la porte (87 aujourd'hui) sont cités avant et après chaque changement.

## Conception

### 1. Les trois preuves, dans les deux modes

Les preuves s'appliquent aux tests d'un comportement déjà en place (`characterize`) **et** aux tests d'une fonctionnalité nouvelle (`green`). En mode nouveau, `red` reste : le test échoue avant le code. Après le commit `feat`, le mutant-writer écrit mutants et changements neutres sur ce commit, et `green` les exécute.

**Dossier de preuves** (hors dépôt, un par slug), option `--proofs DIR` (`--mutants DIR` reste accepté comme synonyme) :

- `<ID>.patch` : mutant de la clause 1 ; `<ID>.2.patch`, `<ID>.3.patch`… : clauses suivantes.
- `<ID>.neutral.patch` : changement neutre du critère.

**Clauses.** Le « Alors » d'un critère est une ligne (une clause) ou une ligne suivie d'une liste (une clause par élément). La porte compte les clauses dans l'entrée de `TESTS.md` et exige un mutant par clause.

**Changement neutre.** Une modification du code de production ou de la configuration après laquelle le « Alors » du critère est toujours vrai : autre valeur valide d'un réglage que le critère lit, élément ajouté à un ensemble qui a vocation à grandir (catalogue, liste d'environnements), ordre changé là où aucun ordre n'est promis, renommage d'un élément interne. Si aucune modification de ce genre n'existe parce que le « Alors » recopie une valeur, le critère est une recopie : le mutant-writer répond `NO-NEUTRAL <ID> : <raison>`, et l'orchestrateur tranche (critère retiré, ou réécrit, ou « ce détail est le contrat » consigné).

**Déroulé commun** (`characterize` sur le commit d'acceptation, `green` sur le commit `feat`) : base verte ; pour chaque critère du slug, chaque mutant est appliqué, la tâche lancée, l'arbre restauré ; puis chaque changement neutre de la même façon ; puis la table de redondance.

**Redondance.** Pendant les mutants, la porte note quels critères (du slug et des sections antérieures jugées par la même tâche) ont un test en échec. Un critère A dont tous les mutants font aussi échouer un test d'un même autre critère B est refusé : B le couvre déjà.

**Code minimal (décision du 10/11, option 2).** Une quatrième preuve porte sur le code, pas sur le test : en `green --proofs`, après les preuves des tests, la porte prend chaque bloc de lignes ajouté par la tâche dans un fichier de production (différence entre le commit d'acceptation et le commit jugé, bloc par bloc), le retire seul, lance la tâche, puis restaure l'arbre. Un bloc dont le retrait ne fait échouer aucun test est refusé : `UNNEEDED <fichier>:<début>-<fin>`. Un bloc refusé à tort (commentaire, message d'erreur sans test) passe par `--unneeded-ok <fichier>:<début>-<fin>` après un `Ruling:`. Limite connue : la preuve ne voit que les blocs entiers. Elle se lance sur le commit d'après refactor.

**Exemption.** `green --proof-exempt` saute les preuves pour une tâche sans changement de comportement (`fix(ci)`, `docs`, `refactor` sous les tests existants) ; comme `--allow-compile-red`, il ne s'emploie qu'après un `Ruling:` consigné.

### 2. Mutant dans `mise.toml`

Un mutant qui touche `mise.toml` est accepté quand toutes ses modifications sont dans des tables `[tasks.…]` autres que la tâche jugée (`--task`), `check`, et toute tâche dont le nom commence par `test`. Le mutant-writer peut lire `mise.toml`.

### 3. Contrôles de forme

| Contrôle | Modes | Refus |
|---|---|---|
| Chaque entrée du slug jugé porte `Étant donné`, `Quand`, `Alors` | `red`, `green`, `characterize` | `INCOMPLETE <ID>: Étant donné/Quand/Alors manquant` |
| Un test dont l'ID existe avant et après, hors `--ac-changed`, est identique au caractère près, même dans un fichier autorisé par `--replaces` | `green`, `characterize` | `KEPT TEST MODIFIED <fichier> <ID>` |
| Rapport absent : le message nomme la tâche et le chemin attendu | les trois | `REPORT missing: <tâche> n'a pas écrit <chemin>` |
| La tâche de la ligne `Commande` du slug est lancée par un hook de commit ou un job CI du dépôt, directement ou par une tâche qui en dépend | `green`, `characterize` | `NOT WIRED <tâche>` |
| La spec passée à `--spec`, quand elle est dans le dépôt, est commitée sans modification en attente | `red`, `characterize` | `SPEC NOT COMMITTED` |

### 4. Rejeu : `tdd-gate replay`

`tdd-gate replay --repo R --base <sha> --head <sha> --proofs DIR --task t [--accept-lost <nom>]...` (une fois par perte acceptée)

Pour chaque mutant du dossier, la porte crée deux copies de travail temporaires (base et tête), applique le mutant de chaque côté, lance les tâches, puis retire les copies. Un mutant est tué d'un côté quand une tâche y sort en erreur.

- `KEPT <nom>` : tué des deux côtés. `NEVER <nom>` : vivant des deux côtés (information).
- `LOST <nom>` : tué sur la base, vivant en tête : la pile a perdu une protection. Échec, sauf `--accept-lost` après un `Ruling:`.

Obligatoire avant la revue finale de toute pile dont une tâche a utilisé `--replaces` ou `--ac-changed`. Les mutants du rejeu sont écrits sur la base par le mutant-writer, à partir du code de production seul (relations et logiques que les profils réels exercent).

### 5. Textes

**Règles génériques (demande du 10/11).** Une règle du harnais s'énonce sans nommer un outil de test, un dépôt ni un chart. Les cas réels servent à trouver la règle et restent cités comme exemples, jamais comme règle. Dans le tableau ci-dessous, les points propres à un outil (helm-unittest) sont donc à écrire sous leur forme générique, l'exemple en « Cas réel » ; une table « cas réel → règle générique » accompagne les textes à la relecture.

**Rouge, vert, refactor (demande du 10/11).** Le déroulé d'une tâche gagne un troisième temps, comme en TDD : après la porte `green`, l'implementer réduit le code de la tâche à son strict minimum (ce qu'aucun test ne demande est retiré), sans toucher aux tests ; commit `refactor(<slug>)` ou réponse `NOTHING TO REDUCE` ; la porte `green` est rejouée sur ce commit avec le même commit d'acceptation. En mode nouveau, mutants et changements neutres s'écrivent sur le commit d'après refactor. Lignes concernées du tableau : `skills/stacked-tdd-mr/SKILL.md`, `agents/implementer.md`.

Aucun critère d'acceptation pour les textes : un test qui vérifierait qu'un fichier contient une phrase serait une recopie. Chaque règle importante est tenue par un contrôle de la porte ; le texte explique comment le passer. Relecture par l'utilisateur.

| Fichier | Changement |
|---|---|
| `templates/TESTS.md` | Entrée avec le critère en toutes lettres (Étant donné / Quand / Alors en sous-liste), plus de renvoi vers la spec |
| `templates/acceptance-criteria.md` | Forme du « Alors » à clauses ; grille unique : quatre raisons de garder (relation entre deux endroits du code, logique, contrat avec l'extérieur, incident réel), six raisons de refuser (teste l'outil, recopie la configuration, fige un détail, cas hypothétique, doublon, ne peut pas échouer), chacune avec la preuve qui la détecte ; règle du catalogue ; règle de plan : une tâche = un fichier de test, la tâche de test écrit un rapport et n'a aucun fichier intermédiaire partagé |
| `agents/acceptance-writer.md` | Critère recopié en toutes lettres dans `TESTS.md` ; un test par critère et par profil, pas davantage ; test de relation en helm-unittest (les deux côtés en littéral dans le même `it`) ; piège `not: true` + `any: true` ; rapport écrit dans le fichier donné par le brief |
| `agents/mutant-writer.md` | Un mutant par clause ; un changement neutre par critère ; constate par un rendu ou une exécution que la sortie change là où la clause le dit (mutant) ou que le « Alors » reste vrai (neutre) ; réponses `NO-MUTANT <ID> clause n`, `NO-NEUTRAL <ID>` ; peut lire `mise.toml` ; mode rejeu (mutants sur la base) |
| `agents/implementer.md` | Rapport écrit dans le fichier donné par le brief |
| `skills/stacked-tdd-mr/SKILL.md` | Mode nouveau : mutant-writer après l'implementer, `green --proofs` ; `characterize --proofs` ; `replay` avant la revue finale ; ordre des commits (`docs(spec)` → `chore(tooling)` → `test(<slug>): acceptance` → `feat` / `fix(ci)`) ; `HEAD` = commit d'acceptation pour `characterize` ; tâche de test branchée avant la première acceptation ; inventaire refait quand la base bouge ; un chiffre n'entre au journal qu'après lecture de la sortie ; procédure de réécriture |
| `skills/repo-tooling` | Gabarit de tâche de test : rapport JUnit, fichiers temporaires propres à la tâche ; `mise run check` rejoué trois fois après un commit d'outillage |

### 6. Réécriture des tests existants (H2)

Procédure ajoutée à la compétence, `Mode: rewrite` : pour un dépôt dont `TESTS.md` a des sections à l'ancien format, une pile dont chaque tâche reprend un fichier de test en `characterize --replaces <anciens slugs>`. Les entrées sont réécrites au nouveau format ; chaque test repasse les trois preuves ; un test qui en échoue une est retiré ou réécrit (`Criterion change:`) ; la pile se termine par `replay` base contre tête.

Cette spec livre la procédure. Son application dépôt par dépôt se planifie dans le chantier d'élagage : d'abord les dépôts qui restent (un dépôt de déploiement, deux dépôts de configuration), ensuite les dépôts déjà élagués (cinq charts, un autre dépôt de déploiement), qui n'ont passé que la preuve « sensible ».

### Contraintes

- Les 87 tests actuels de la porte restent verts à chaque étape ; chaque contrôle nouveau arrive avec ses tests dans `bin/tests/test_tdd_gate.py`, écrits et vus rouges avant le code.
- Les sections de `TESTS.md` autres que le slug jugé ne sont soumises à aucun contrôle nouveau, sauf « test gardé inchangé » et la redondance, qui les lisent sans les juger.
- Coût accepté : une exécution de mutant-writer de plus par tâche en mode nouveau ; durée de la porte environ doublée (un lancement de tâche par mutant et par changement neutre).

### Non retenu, faute de cas réel

- Vérifier que le nom de test cité après `›` existe.
- Comparer le texte du critère entre `TESTS.md` et la spec.
- Détecter le test de l'outil en lançant les linters sous mutant.

## Critères d'acceptation

Frontière de tous les critères : la ligne de commande `tdd-gate` sur un dépôt git jetable (sortie standard et code de sortie).

### PRV-001 — Un mutant par clause est exigé
- **Étant donné** un slug dont le critère `X-001` a un « Alors » à deux clauses, et un dossier de preuves qui contient `X-001.patch` et `X-001.neutral.patch` mais pas `X-001.2.patch`
- **Quand** `tdd-gate characterize` est lancé sur le commit d'acceptation
- **Alors** la sortie contient `MUTANT MISSING X-001 clause 2` et se termine par `FAIL`
- **Cas réel** : un critère d'un dépôt de déploiement, clause « politique restricted » non visée par le mutant

### PRV-002 — Une clause qu'aucun test ne défend est refusée
- **Étant donné** le même critère à deux clauses, les deux mutants présents, et un test qui n'échoue que sous le premier
- **Quand** `tdd-gate characterize` est lancé
- **Alors** la sortie contient `KILLED X-001 clause 1` et `SURVIVED X-001 clause 2`, et se termine par `FAIL`
- **Cas réel** : un critère d'un dépôt de déploiement, clause déjà imposée par le schéma

### PRV-003 — Un changement neutre par critère est exigé
- **Étant donné** un critère `X-001` dont le mutant est présent et tué, sans `X-001.neutral.patch`
- **Quand** `tdd-gate characterize` est lancé
- **Alors**
  - la sortie contient `NEUTRAL MISSING X-001` et se termine par `FAIL`
  - relancée avec un `X-001.neutral.patch` qui ne s'applique pas au dépôt, elle contient `NEUTRAL INVALID X-001` et se termine par `FAIL`
- **Cas réel** : changement neutre qui ne s'applique pas, vu le 10/11 sur trois dépôts de test du harnais (`NEUTRAL INVALID CALC-003: patch does not apply`) ; 164 entrées « recopie de la configuration » dans les `scope.psv` du chantier

### PRV-004 — Un test qui casse sous un changement neutre est refusé
- **Étant donné** un critère `X-001` dont le test affirme le nom d'un élément du catalogue, et un changement neutre qui renomme cet élément
- **Quand** `tdd-gate characterize` est lancé
- **Alors** la sortie contient `BRITTLE X-001` avec le nom du test en échec, et se termine par `FAIL`
- **Cas réel** : un critère d'un dépôt de déploiement nommait une application du catalogue

### PRV-005 — Un test sensible et spécifique passe
- **Étant donné** un critère `X-001` dont le « Alors » a deux clauses en sous-liste, suivie de la ligne `- **Test**`, ses deux mutants, son changement neutre, et des tests qui échouent sous chaque mutant et passent sous le changement neutre
- **Quand** `tdd-gate characterize` est lancé
- **Alors** la sortie contient `KILLED X-001 clause 1`, `KILLED X-001 clause 2` et `NEUTRAL OK X-001`, aucune ligne `MUTANT MISSING`, et se termine par `PASS`

### PRV-006 — Un critère couvert par un autre est refusé
- **Étant donné** deux critères `X-001` et `X-002` jugés par la même tâche, où le mutant de `X-002` fait aussi échouer le test de `X-001`, et où le mutant de `X-001` ne fait pas échouer le test de `X-002`
- **Quand** `tdd-gate characterize` est lancé
- **Alors** la sortie contient `REDUNDANT X-002` en nommant `X-001`, ne contient pas `REDUNDANT X-001`, et se termine par `FAIL`
- **Cas réel** : un critère d'un dépôt de déploiement, jugé doublon à la main ; 100 entrées « doublon » dans les `scope.psv`

### PRV-007 — Les preuves valent aussi pour une fonctionnalité nouvelle
- **Étant donné** un commit d'acceptation et un commit `feat` qui rend ses tests verts
- **Quand** `tdd-gate green` est lancé sans `--proofs` ni `--proof-exempt`
- **Alors**
  - la sortie contient `PROOF MISSING` et se termine par `FAIL`
  - relancé avec `--proofs` et un test qui survit à son mutant, elle contient `SURVIVED` et se termine par `FAIL`
  - relancé avec `--proofs`, un test qui échoue sous son mutant et passe sous son changement neutre, elle contient `KILLED` et `NEUTRAL OK` pour le critère, aucune ligne `SURVIVED`, et se termine par `PASS`
  - relancé avec `--proof-exempt`, aucun mutant n'est appliqué et elle se termine par `PASS`
- **Cas réel** : en mode nouveau, la porte `green` n'applique aucun mutant aujourd'hui (`tdd-gate:523-592`)

### PRV-008 — Un mutant du code écrit dans une tâche mise est jugé
- **Étant donné** un dépôt dont le code testé est le corps de la tâche `[tasks."bootstrap:agent"]` de `mise.toml`, jugé par la tâche `test:bootstrap`
- **Quand** `tdd-gate characterize` reçoit un mutant qui modifie le corps de `bootstrap:agent`, puis un mutant qui modifie le corps de `test:bootstrap`
- **Alors**
  - le premier est appliqué et jugé (`KILLED` ou `SURVIVED`)
  - le second est refusé par `MUTANT INVALID … touches test task test:bootstrap`
  - un mutant d'un autre fichier que `mise.toml`, dont le hunk porte sur des numéros de ligne qui sont, dans `mise.toml`, ceux du corps de `test:bootstrap`, est appliqué et jugé (`KILLED` ou `SURVIVED`)
- **Cas réel** : un dépôt de déploiement, 8 mutants sur 9 refusés pour la tâche d'amorçage ; `mise.toml` de 299 lignes (`test:bootstrap` ligne 62) et mutants de gabarits Helm à des lignes quelconques

### PRV-009 — Un bloc de code qu'aucun test ne demande est refusé
- **Étant donné** un commit d'acceptation et un commit `feat` qui ajoute, dans un fichier de production, deux blocs séparés : l'un sans lequel le test du critère échoue, l'autre dont le retrait laisse tous les tests verts
- **Quand** `tdd-gate green --proofs` est lancé
- **Alors**
  - la sortie contient `UNNEEDED` avec le fichier et les lignes du second bloc, aucune ligne `UNNEEDED` pour le premier, et se termine par `FAIL`
  - relancé avec `--unneeded-ok` pour ce bloc, elle contient `UNNEEDED ACCEPTED` avec le même fichier et les mêmes lignes, aucune autre ligne `UNNEEDED`, et se termine par `PASS`
  - quand le commit `feat` supprime aussi une ligne existante d'un fichier de production et ajoute un fichier de test, aucune ligne `UNNEEDED` ne nomme la suppression ni le fichier de test
- **Cas réel** : demande du 10/11, « une étape de refactor (comme en vrai TDD), pour réduire le code à son strict minimum », option 2 retenue (preuve par retrait dans la porte)

### PRV-010 — Un changement neutre sans résultat est invalide
- **Étant donné** un critère `X-001` dont le mutant est tué, et un `X-001.neutral.patch` qui s'applique mais sous lequel la tâche de test ne produit aucun résultat pour les tests du critère
- **Quand** `tdd-gate characterize` est lancé
- **Alors** la sortie contient `NEUTRAL INVALID X-001`, aucune ligne `NEUTRAL OK X-001`, et se termine par `FAIL`
- **Cas réel** : un chart, `MUTANT INVALID ID-003: no result` sur une panne réseau du téléchargement des dépendances Helm ; la même panne pendant le changement neutre donnait `NEUTRAL OK`

### FRM-001 — Le critère est écrit en toutes lettres dans l'entrée
- **Étant donné** un `TESTS.md` dont l'entrée `X-001` du slug jugé ne porte qu'un renvoi (`**Critère** : docs/specs/x.md › X-001`), et dont une section d'un autre slug est à l'ancien format
- **Quand** `tdd-gate red`, `green` ou `characterize` est lancé sur le slug
- **Alors** la sortie contient `INCOMPLETE X-001` et se termine par `FAIL`, sans aucune ligne `INCOMPLETE` pour l'autre section
- **Cas réel** : demande du 10/11 ; `templates/TESTS.md:7`

### FRM-007 — Une tâche atteinte par un motif de dépendance est branchée
- **Étant donné** un slug dont la `Commande` est `mise run test:render`, un hook de commit qui lance `mise run test`, et un `mise.toml` où `[tasks.test]` a `depends = ["test:*"]`
- **Quand** `tdd-gate green` ou `characterize` est lancé
- **Alors** la sortie ne contient aucune ligne `NOT WIRED`
- **Cas réel** : un dépôt de CI (`mise.toml:92-94`, hook `mise run test`) ; gabarit `skills/repo-tooling/templates/mise.toml:33-35`

### PRV-011 — Un mutant sans en-tête `diff --git` est jugé
- **Étant donné** un dossier de preuves dont le mutant de `X-001` est un patch qui commence par `--- a/<fichier>`, sans ligne `diff --git`, et qui casse le test du critère
- **Quand** `tdd-gate characterize` est lancé
- **Alors** la sortie contient `KILLED X-001`
- **Cas réel** : un mutant d'un chantier antérieur, jugé `KILLED` le 02/11, `UnboundLocalError` avec la porte du 11/11

### PRV-012 — Les preuves ne s'appliquent pas sur un arbre modifié
- **Étant donné** un dépôt dont un fichier suivi porte une modification non commitée
- **Quand** `tdd-gate green --proofs DIR` est lancé
- **Alors** la sortie contient `TREE NOT CLEAN`, aucune ligne `MUTANT`, se termine par `FAIL`, et la modification non commitée est toujours dans le fichier
- **Cas réel** : 10/11, implémentation laissée non commitée avant une porte verte (registre du harnais déterministe)

### PRV-013 — Le code minimal se juge depuis le début de la tâche
- **Étant donné** un slug qui a deux commits d'acceptation, et un bloc de code de production qu'aucun test ne demande, ajouté entre les deux
- **Quand** `tdd-gate green --proofs DIR` est lancé avec `--test <second commit d'acceptation>` et `--since <premier commit d'acceptation>`
- **Alors** la sortie contient `UNNEEDED` avec le fichier et les lignes de ce bloc, et se termine par `FAIL`
- **Cas réel** : auto-preuve du 11/11, slug `rejeu` : le mode `replay` (`db779fa`), écrit avant le second commit d'acceptation, n'a pas été jugé

### PRV-014 — La spec n'est pas jugée comme du code
- **Étant donné** un slug qui a deux commits d'acceptation et, entre les deux, un commit qui ajoute des lignes au fichier de spec du dépôt
- **Quand** `tdd-gate green --proofs --since <premier commit d'acceptation> --spec <fichier de spec>` est lancé
- **Alors** la sortie ne contient aucune ligne `UNNEEDED` qui nomme le fichier de spec, et se termine par `PASS`
- **Cas réel** : auto-preuve du 11/11, slug `mutant-mise`, après le changement de critère PRV-008 : `UNNEEDED docs/specs/2026-11-10-harnais-deterministe-design.md:170-171`

### PRV-015 — Un test gardé n'est pas dit modifié quand son voisin change
- **Étant donné** un fichier de test où le test du critère `X-001` est suivi d'un autre test, et un commit d'acceptation qui laisse le test de `X-001` intact au caractère près
- **Quand** `tdd-gate characterize` est lancé sans `--ac-changed`
- **Alors**
  - quand le test qui suit est un test pytest précédé d'un décorateur (ligne `@…` au-dessus de son nom) et que le commit le retire, si bien que le test suivant, précédé d'un autre décorateur, prend sa place, la sortie ne contient aucune ligne `KEPT TEST MODIFIED` qui nomme `X-001`
  - quand le fichier est un fichier helm-unittest à plusieurs documents et que le commit retire le document qui suit le test de `X-001` (séparateur `---` puis en-tête `suite:`), la sortie ne contient aucune ligne `KEPT TEST MODIFIED` qui nomme `X-001`
- **Cas réel** : deux branches d'élagage d'un chart (décorateur `@pytest.mark.parametrize` d'un test retiré ; `---` et en-tête d'une suite retirée), rejugement du 11/11

### PRV-016 — Un test gardé devenu le dernier du fichier n'est pas dit modifié
- **Étant donné** un fichier de test pytest qui se termine par un bloc `if __name__ == "__main__":`, où le test du critère `X-001` est suivi d'un dernier test, et un commit d'acceptation qui retire ce dernier test en laissant le test de `X-001` intact au caractère près
- **Quand** `tdd-gate characterize` est lancé sans `--ac-changed`
- **Alors** la sortie ne contient aucune ligne `KEPT TEST MODIFIED` qui nomme `X-001`
- **Cas réel** : un dépôt de déploiement, rejugement du 11/11 (le test retiré était le dernier du fichier)

### RPL-003 — Une tâche qui échoue sans mutant arrête le rejeu
- **Étant donné** deux commits dont le premier fait échouer la tâche sans aucun mutant
- **Quand** `tdd-gate replay --base <premier> --head <second>` est lancé
- **Alors**
  - la sortie contient `BASELINE NOT GREEN base`, aucune ligne `KEPT`, et se termine par `FAIL`
  - relancé en échangeant `--base` et `--head`, elle contient `BASELINE NOT GREEN head`, aucune ligne `KEPT`, et se termine par `FAIL`
- **Cas réel** : un chart, panne réseau du téléchargement des dépendances Helm : la tâche échoue des deux côtés et tous les mutants paraîtraient gardés

### FRM-006 — L'entrée a autant de clauses que la spec
- **Étant donné** une spec dont le critère `X-001` a un « Alors » en liste de deux clauses, et un `TESTS.md` dont l'entrée `X-001` du slug jugé porte ce « Alors » sur une seule ligne
- **Quand** `tdd-gate red` ou `characterize` est lancé avec `--spec`
- **Alors** la sortie contient `CLAUSES X-001: 1 in TESTS.md, 2 in spec` et se termine par `FAIL`
- **Cas réel** : entrée FRM-002 de ce dépôt au commit `f1fd102`, trois clauses de la spec recopiées sur une ligne, une seule clause prouvée par mutant

### FRM-002 — Un test gardé ne change pas
- **Étant donné** un fichier de test cité par un slug remplacé (`--replaces`), où le test `X-001`, présent dans `TESTS.md` avant et après, a une ligne modifiée, et où `X-002` est modifié et listé dans `--ac-changed`
- **Quand** `tdd-gate characterize` est lancé
- **Alors**
  - la sortie contient `KEPT TEST MODIFIED` avec le fichier et `X-001`, aucune ligne de ce type pour `X-002`, et se termine par `FAIL`
  - quand `X-001` est laissé intact et qu'un test de `X-002` est ajouté à la fin du fichier, après une ligne vide, la sortie ne contient aucune ligne `KEPT TEST MODIFIED`
  - quand `X-001` est porté par deux tests du fichier, tous deux laissés intacts, la sortie ne contient aucune ligne `KEPT TEST MODIFIED`
- **Cas réel** : comparaison bloc par bloc faite à la main à chaque tâche de l'élagage

### FRM-003 — Un rapport absent dit quoi faire
- **Étant donné** une tâche de test qui sort à 0 sans écrire de rapport
- **Quand** `tdd-gate characterize` est lancé
- **Alors** la sortie contient une ligne `REPORT missing` qui nomme la tâche et le chemin `.tdd/junit.xml`, et se termine par `FAIL`
- **Cas réel** : `test:render` et `test:bootstrap` d'un dépôt de déploiement

### FRM-004 — La tâche de test tourne quelque part
- **Étant donné** un slug dont la `Commande` est `mise run test:render`, tâche citée par aucun hook de commit ni job CI du dépôt, ni dépendance d'une tâche qui l'est
- **Quand** `tdd-gate characterize` ou `green` est lancé
- **Alors**
  - la sortie contient `NOT WIRED test:render` et se termine par `FAIL`
  - une fois la tâche citée dans `.pre-commit-config.yaml` ou `.gitlab-ci.yml`, la ligne `NOT WIRED` n'apparaît plus
  - quand le hook ne cite que `mise run check` et que `test:render` est une dépendance (`depends`) de `check` dans `mise.toml`, la ligne `NOT WIRED` n'apparaît pas
- **Cas réel** : un dépôt de déploiement (`test:render`, `test:bootstrap`), trois charts d'application ; dépendance : le dépôt du harnais lui-même (hook `mise run check`, `check` dépend de `test`)

### FRM-005 — La spec jugée est celle qui est commitée
- **Étant donné** une spec située dans le dépôt, avec une modification non commitée
- **Quand** `tdd-gate red` ou `characterize` est lancé avec `--spec`
- **Alors** la sortie contient `SPEC NOT COMMITTED` et se termine par `FAIL`
- **Cas réel** : deux critères, commit de spec inséré après coup

### RPL-001 — Une protection perdue par la pile est signalée
- **Étant donné** deux commits d'un dépôt, un mutant qui fait échouer la tâche sur le premier et la laisse passer sur le second
- **Quand** `tdd-gate replay --base <premier> --head <second>` est lancé
- **Alors**
  - la sortie contient `LOST` avec le nom du mutant et se termine par `FAIL`
  - relancé avec `--accept-lost <nom>`, elle contient `LOST ACCEPTED` et se termine par `PASS`
  - relancé en échangeant `--base` et `--head`, elle contient `GAINED` avec le nom du mutant et se termine par `PASS`
- **Cas réel** : un chart, test retiré à tort (ExternalSecret et Deployment) ; pertes dans quatre charts sur cinq

### RPL-002 — Un rejeu sans perte passe et ne laisse rien derrière lui
- **Étant donné** deux commits et deux mutants, l'un tué des deux côtés, l'autre vivant des deux côtés
- **Quand** `tdd-gate replay` est lancé
- **Alors**
  - la sortie contient `KEPT` pour le premier et `NEVER` pour le second, et se termine par `PASS`
  - après l'exécution, `git worktree list` et `git status --short` du dépôt sont ceux d'avant

### RPL-004 — Plusieurs pertes acceptées se nomment chacune
- **Étant donné** deux commits d'un dépôt et deux mutants, chacun fait échouer la tâche sur le premier et la laisse passer sur le second
- **Quand** `tdd-gate replay --base <premier> --head <second>` est lancé avec `--accept-lost <mutant 1> --accept-lost <mutant 2>`
- **Alors**
  - la sortie contient `LOST ACCEPTED` avec le nom de chacun des deux mutants et se termine par `PASS`
  - relancé avec le seul `--accept-lost <mutant 1>`, elle contient `LOST` avec le nom du mutant 2 et se termine par `FAIL`
- **Cas réel** : rejeu de fin de pile d'un dépôt de déploiement, 12 pertes acceptées sur une tâche et 4 sur une autre ; avec `--accept-lost` répété, seul le dernier nom était retenu et le rejeu finissait en `FAIL`

### LOC-001 — Un hôte déclaré en local donne son outil de forge
- **Étant donné** un dépôt git dont `origin` vaut `https://forge.exemple.test/groupe/depot.git`, et un dossier local (variable `CLAUDE_LOCAL_DIR`) dont le fichier `forges` contient la ligne `forge.exemple.test glab`
- **Quand** `harness-local forge --repo <dépôt>` est lancé
- **Alors** la sortie est `glab` et le code de sortie est 0
- **Cas réel** : deux forges internes retirées de la version publique du skill le 11/11 (demande de l'utilisateur : le concret vit hors Git)

### LOC-002 — Sans fichier local, seuls les hôtes publics sont connus
- **Étant donné** un dossier local vide (variable `CLAUDE_LOCAL_DIR`)
- **Quand** `harness-local forge --repo <dépôt>` est lancé
- **Alors**
  - pour un dépôt dont `origin` vaut `git@github.com:groupe/depot.git`, la sortie est `gh` et le code de sortie est 0
  - pour un dépôt dont `origin` vaut `https://gitlab.com/groupe/depot.git`, la sortie est `glab` et le code de sortie est 0
  - pour un dépôt dont `origin` vaut `https://forge.exemple.test/groupe/depot.git`, le code de sortie est 1 et la sortie contient `hôte forge.exemple.test inconnu`
- **Cas réel** : version publique du skill, `gitlab.com) forge=glab` et `github.com) forge=gh` ; arrêt « hôte inconnu » déjà prévu par le skill

### LOC-003 — Un cas réel local affine une règle sans être versionné
- **Étant donné** un dossier local (variable `CLAUDE_LOCAL_DIR`) dont le fichier `cas-reels.md` contient une ligne `| PRV-015 | dépôt interne, critère X-103 |`
- **Quand** `harness-local cas-reels PRV-015` est lancé
- **Alors**
  - la sortie contient la ligne du fichier qui nomme `PRV-015`, et le code de sortie est 0
  - quand le fichier `cas-reels.md` n'existe pas, la sortie est vide et le code de sortie est 0
- **Cas réel** : les lignes « Cas réel » de la spec et de la table des règles, neutralisées avant publication le 11/11 ; les relecteurs doivent pouvoir retrouver l'exemple concret d'une règle
