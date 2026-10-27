# Carteira de Documentos (Project LAB, Grupo 3)

Carteira digital de documentos com verificação de integridade. Backend em Go servindo a API e o frontend React embutido no binário.

- Produção: https://carteira.eflow.space (nginx → `127.0.0.1:7546`)
- Quadro: Chronos, "Project LAB Repo (Grupo 3)"

## Rodando

```bash
cp .env.example .env   # preencha Postgres e MinIO
make install           # dependências do frontend
make build             # frontend → backend/web/dist, depois o binário Go
./backend/bin/carteira
```

Desenvolvimento com hot reload: `make dev-api` (porta 7546) e `make dev-web` (Vite em 5173, com proxy de `/api`).

Em produção o processo roda no PM2 (`ecosystem.config.js`):

```bash
make build && pm2 restart carteira
pm2 logs carteira
```

## Documentação

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): visão geral, pastas e modelo de dados
- [docs/API.md](docs/API.md): endpoints com exemplos
- [docs/ATORES.md](docs/ATORES.md) / [docs/CASOS_DE_USO.md](docs/CASOS_DE_USO.md): atores, matriz e casos de uso
- [docs/REQUISITOS.md](docs/REQUISITOS.md) / [docs/REGRAS_NEGOCIO.md](docs/REGRAS_NEGOCIO.md) / [docs/MODULOS.md](docs/MODULOS.md): requisitos, RN e módulos
- [docs/UML.md](docs/UML.md) / [docs/FLUXOS.md](docs/FLUXOS.md) / [docs/GLOSSARIO.md](docs/GLOSSARIO.md): UML, fluxos e glossário
- [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md): mapa RF → código e convenções
- [docs/PRIVACIDADE.md](docs/PRIVACIDADE.md) / [docs/TERMOS.md](docs/TERMOS.md): LGPD — controlador pessoa física Irving Samuel Lima Bandeira (`/privacidade`, `/termos`)
