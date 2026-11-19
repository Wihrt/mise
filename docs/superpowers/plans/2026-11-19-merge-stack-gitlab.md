# `harness merge-stack` sur GitLab — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: `stacked-tdd-mr` (wrapper over superpowers:subagent-driven-development). Steps use checkbox (`- [ ]`) syntax for tracking. One task = one branch = one PR, stacked.

**Goal:** `harness merge-stack <MR du haut>` fusionne et prouve une pile sur GitLab comme sur GitHub, et une pile peut viser une autre branche que la branche par défaut.

**Architecture:** Un adaptateur `harness/internal/adapters/gitlab` derrière le port `app.Forge` existant, choisi dans `main.go` par l'hôte de `origin` via la correspondance de `harness local forge`. Le cœur `app.MergeStack` ne change que sur la fin de la pile (cible = première branche qui n'est la tête d'aucune MR ouverte, avec une garde).

**Tech Stack:** Go (module `github.com/Wihrt/mise/harness`), `glab api`, bats, jq.

**Spec:** `docs/superpowers/specs/2026-11-19-merge-stack-gitlab-design.md`

## Global Constraints

- Dépôt public : aucun hôte interne ni nom de dépôt réel ; hôtes de test `forge.example.test`, `inconnu.example.test`.
- L'adaptateur GitLab n'appelle que `glab api` (projet `:id`), jamais `glab mr merge`.
- Fusion GitLab : `PUT projects/:id/merge_requests/<iid>/merge` avec `sha=<sha vérifié>` et `should_remove_source_branch=true`.
- Lignes de sortie identiques sur les deux forges (`#n` pour le numéro) ; là où elles nommaient la branche par défaut, elles nomment la cible de la pile.
- Délais : `HARNESS_POLL_INTERVAL` (15s), `HARNESS_CHECKS_GRACE` (60s), `HARNESS_CHECKS_TIMEOUT` (30m), partagés par les deux adaptateurs.
- Les tests FUS-001 à FUS-009 existants restent verts sans modification de leurs assertions.
- Aucun test d'acceptation dans ce plan : `acceptance-writer` les écrit depuis la section « Critères d'acceptation » de la spec.

## Review Focus

1. Une MR GitLab dont `detailed_merge_status` reste transitoire (`checking`, `unchecked`, `preparing`, `ci_still_running`) au-delà de `HARNESS_CHECKS_TIMEOUT` : erreur nommée, `STOPPED`, pas d'attente infinie. → test unitaire Task 2.
2. `glab api --paginate` sur la recherche par branche : une MR sur la deuxième page doit être vue par la garde. → test unitaire Task 3.
3. Référence de MR donnée comme URL d'un autre projet ou comme `!12` : la première refusée, la seconde acceptée. → test unitaire Task 2.
4. `origin` en SSH (`git@forge.example.test:g/d.git`) : même choix de forge qu'en HTTPS. → test unitaire Task 1.
5. Après repointage par GitLab, la MR du dessus a déjà `target_branch` = cible : le repointage explicite ne doit pas échouer. → test unitaire Task 2.

---

### Task 1: Choix de la forge par l'hôte de `origin`

**Critères :** FGL-010 (et FUS-001…009 restent verts).

**Files:**
- Modify: `harness/internal/app/local/local.go` (extraire la résolution hôte → outil de `Forge`)
- Modify: `harness/cmd/harness/main.go:80-105` (`runMergeStack` : choisir l'adaptateur avant tout appel)
- Test (unitaire) : `harness/internal/app/local/local_test.go`

**Interfaces:**
- Produces: `func (l *Local) Tool(repo string) (tool string, host string, err error)` dans `local` — renvoie `"gh"` ou `"glab"` ; hôte absent de `forges` et des défauts publics → `err` dont le texte nomme l'hôte et le chemin `<CLAUDE_LOCAL_DIR>/forges`. `Forge(repo)` l'utilise et garde sa sortie actuelle.
- Produces: dans `main.go`, `newForge(tool, dir string, t Timing) (app.Forge, error)` ; `"glab"` renvoie une erreur `glab: not supported yet` jusqu'à la Task 2. Erreur → `FAIL <message>` sur stdout, code 1, aucun appel à `gh`/`glab`.
- Produces: type `Timing` déplacé dans un endroit partagé par les deux adaptateurs (`harness/internal/adapters/forgetiming` ou `app`), au choix de l'implémenteur ; `github.Timing` devient un alias ou disparaît.

- [ ] **Step 1:** Test unitaire `TestTool` : `https://forge.example.test/g/d.git` et `git@forge.example.test:g/d.git` avec `forges` = `forge.example.test glab` → `glab` ; `https://github.com/o/r.git` sans `forges` → `gh` ; `https://inconnu.example.test/g/d.git` → erreur contenant `inconnu.example.test` et `forges`.
- [ ] **Step 2:** `go test ./internal/app/local/` → FAIL (`Tool` non défini).
- [ ] **Step 3:** Implémenter `Tool`, réécrire `Forge` dessus, brancher `runMergeStack` (lecture de `origin` par l'adaptateur `git` déjà utilisé par `runLocal`).
- [ ] **Step 4:** `mise run test` → PASS (FUS-* compris).
- [ ] **Step 5:** Commit `feat(merge-stack-gitlab): le choix de la forge suit l'hôte de origin`.

### Task 2: Adaptateur GitLab

**Critères :** FGL-001, FGL-002, FGL-003, FGL-004, FGL-005, FGL-006, FGL-007, FGL-008, FGL-009.

**Files:**
- Create: `harness/internal/adapters/gitlab/forge.go`
- Test (unitaire) : `harness/internal/adapters/gitlab/forge_test.go` (glab simulé par un exécutable de test sur le `PATH`, comme bon semble à l'implémenteur)
- Modify: `harness/cmd/harness/main.go` (`newForge("glab", …)` → `gitlab.NewForge`)
- Modify: `harness/internal/app/mergestack.go` (`ready` : refus `CONFLICTS #n` et `NOT MERGEABLE #n: <statut>` — voir Interfaces)
- Modify: `harness/internal/domain/stack.go` (champs de `PullRequest`)
- Modify: `claude/CLAUDE.md` (retirer « On GitLab the command does not exist yet: hand over the MR links. »)
- Modify: `claude/skills/forge-glab/SKILL.md` (ajouter, comme `forge-gh` : « When the user asks for the merge: `harness merge-stack <top MR>`, never `glab mr merge` (`stacked-tdd-mr`, "Merging a stack"). »)
- Modify: `claude/skills/stacked-tdd-mr/SKILL.md` (section « Merging a stack » : retirer toute mention que GitLab n'est pas pris en charge)
- Le faux `claude/hooks/tests/fake-forge/glab` est écrit par `acceptance-writer` avec les tests.

**Interfaces:**
- Consumes: `newForge`, `Timing` (Task 1).
- Produces: `func NewForge(dir string, t Timing) *Forge` dans `gitlab`, qui implémente `app.Forge`.
- Produces: `domain.PullRequest` gagne `Conflicts bool` et `MergeStatus string` (vide = la forge ne le dit pas, cas GitHub). `ready` ajoute, après les contrôles `NOT OPEN`/`DRAFT` : `Conflicts` → `CONFLICTS #n` ; `MergeStatus != "" && MergeStatus != "mergeable"` → `NOT MERGEABLE #n: <MergeStatus>`. Les deux mettent `ok = false`.
- Comportement de chaque méthode : section « Adaptateur `gitlab` » et « Points tranchés » de la spec, à suivre telle quelle (endpoints, champs, sources `merge_request_event`, états transitoires dont `ci_still_running`, état `merged` relu après la fusion).
- `Get(ref)` : `12`, `!12`, `<web_url du projet>/-/merge_requests/12` acceptés ; URL d'un autre projet → erreur ; sinon nom de branche → `ByHead`.

- [ ] **Step 1:** Tests unitaires de l'adaptateur : correspondance des champs de `merge_requests/<iid>` vers `PullRequest` ; `Get("!12")` et URL d'un autre projet (Review Focus 3) ; `Checks` vert seulement si `head_pipeline.sha` = `sha` ; `Merge` attend un statut transitoire puis fusionne, et renvoie une erreur nommée au timeout (Review Focus 1) ; `Retarget` sur une MR déjà repointée ne renvoie pas d'erreur (Review Focus 5). Test de `ready` : `CONFLICTS #2`, `NOT MERGEABLE #2: not_approved`.
- [ ] **Step 2:** `go test ./internal/adapters/gitlab/ ./internal/app/` → FAIL.
- [ ] **Step 3:** Implémenter l'adaptateur, les champs de domaine, `ready`, le branchement dans `main.go` ; mettre à jour les trois fichiers de doc.
- [ ] **Step 4:** `mise run test` → PASS.
- [ ] **Step 5:** Commit `feat(merge-stack-gitlab): harness merge-stack fusionne les piles GitLab`.

### Task 3: Cible de la pile et garde

**Critères :** FGL-011, FGL-012.

**Files:**
- Modify: `harness/internal/app/mergestack.go` (`resolve`, `ready`, `merge`, `prove`, port `Forge`)
- Modify: `harness/internal/adapters/github/forge.go`, `harness/internal/adapters/gitlab/forge.go`
- Modify: `harness/internal/domain/stack.go` (commentaire de `Stack` : « the first targets the target of the stack »)
- Test (unitaire) : `harness/internal/app/mergestack_test.go` (existe ; forge en mémoire)
- Le faux `gh` (`--state` aujourd'hui ignoré) est adapté par `acceptance-writer`.

**Interfaces:**
- Consumes: `gitlab.Forge` (Task 2).
- Produces: port `Forge` gagne `WasHead(branch string) (domain.PullRequest, bool, error)` — une MR du dépôt (pas d'un fork), dans n'importe quel état, dont la branche source est `branch`. GitHub : `gh pr list --head <b> --state all --limit 1000 --json …`, filtrée sur le dépôt de base ; GitLab : `merge_requests?source_branch=<b>` sans filtre d'état, `--paginate` (Review Focus 2).
- `resolve` renvoie `(target string, stack domain.Stack, err error)` : descend tant que `ByHead(pr.Base)` trouve une MR ouverte ; la `Base` de la MR du bas est la cible. Puis `WasHead(target)` : trouvée → erreur `<target> is the branch of #<n> (<state>); retarget #<bas> on the branch it should land on, then ask again` ; erreur de requête → erreur. Plus de lecture de `DefaultBranch` dans `resolve`.
- `ready`, `merge`, `prove` reçoivent la cible à la place de `def` ; les textes de sortie sont inchangés à la valeur près.
- `DefaultBranch` reste dans le port (non utilisé par `merge-stack` après ce changement : le retirer du port et des deux adaptateurs s'il n'a plus d'appelant).

- [ ] **Step 1:** Tests unitaires de `MergeStack` sur une forge en mémoire : pile vers `dev` → cible `dev` ; `WasHead` trouvée → `FAIL` sans `Merge` ni `Retarget` ; `WasHead` en erreur → `FAIL`. Test GitLab : `WasHead` voit une MR en page 2.
- [ ] **Step 2:** `go test ./...` → FAIL.
- [ ] **Step 3:** Implémenter.
- [ ] **Step 4:** `mise run test` → PASS (FUS-* et FGL-001…010 compris).
- [ ] **Step 5:** Commit `feat(merge-stack-gitlab): une pile vise sa propre cible, jamais une branche abandonnée`.

## Après la pile (hors code)

- Mettre à jour la mémoire `fusion-gitlab-script-utilisateur` : `harness merge-stack` remplace le script.
