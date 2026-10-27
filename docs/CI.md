# CI completo e fluxo Git — Project LAB

Skill do agente Cursor: [`.cursor/skills/ci-completo/SKILL.md`](../.cursor/skills/ci-completo/SKILL.md).  
Comando local/CI: `make ci`.

## Branches

| Branch | Papel | Proteção sugerida no GitHub |
|---|---|---|
| `main` | Produção | Protegida: exigir PR, reviews, status check `CI / make ci`, sem force-push |
| `dev` | Integração | Idem |
| `release/X.Y` | Hotfix / linha pública de versão | CI em push/PR; PRs abertos; force-push desaconselhado |
| `feature/*` | Trabalho | Livres |

### Fluxo

1. Issue descreve a tarefa.  
2. Branch `feature/…` a partir de `dev`.  
3. PR → `dev` (CI verde).  
4. PR `dev` → `main` (CI verde).  
5. Tag `vX.Y.Z` + **GitHub Release**.  
6. Opcional: branch `release/X.Y` a partir da tag para patches.

## Issues

- Um Issue por mudança relevante (bug, RF, doc).  
- Labels sugeridas: `bug`, `enhancement`, `docs`, `ci`.  
- PRs referenciam o Issue (`Closes #N`).

## Pull Requests

- Título curto; corpo com o *porquê*.  
- CI obrigatório: workflow [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).  
- Sem merge com CI vermelho.

## Releases e versions

- SemVer: `vMAJOR.MINOR.PATCH`.  
- Release notes: o que mudou para o usuário/equipe.  
- Hotfix: commit em `release/X.Y` → PR → tag patch (`v1.2.1`) → espelhar em `dev`/`main`.

## Checklist GitHub (quando o remoto existir)

1. Criar branches `main` e `dev` (e a primeira `release/` se já houver versão).  
2. Settings → Branches → Branch protection rules para `main` e `dev`:  
   - Require a pull request before merging  
   - Require status checks to pass → `CI / make ci`  
   - Restrict force pushes  
3. Actions habilitadas no repositório.  
4. (Opcional) Environments `production` para releases.

## Comandos

```bash
make ci       # front build + go vet + go build
make build    # binário + SPA embed
make deploy   # só no host de deploy, após merge
```
