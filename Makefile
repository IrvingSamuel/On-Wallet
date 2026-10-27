.PHONY: install frontend backend build ci run deploy dev-api dev-web

install:
	cd frontend && npm install

frontend:
	rm -rf backend/web/dist/assets backend/web/dist/index.html
	cd frontend && npm run build

# Compila para um arquivo novo e só depois substitui o binário (evita "text file busy" com o PM2).
backend:
	cd backend && GOTOOLCHAIN=go1.26.0 go build -buildvcs=false -o bin/carteira.new ./cmd/server
	mv -f backend/bin/carteira.new backend/bin/carteira

build: frontend backend

# CI completo: typecheck/build do front + vet + build do backend (sem redeploy).
ci:
	$(MAKE) frontend
	cd backend && GOTOOLCHAIN=go1.26.0 go vet ./...
	cd backend && GOTOOLCHAIN=go1.26.0 go build -buildvcs=false -o /tmp/carteira-ci ./cmd/server
	@echo "CI OK"

# Redeploy: para o processo, troca o binário e sobe de novo.
deploy: build
	pm2 stop carteira
	pm2 start carteira

run: build
	./backend/bin/carteira

# Desenvolvimento: API em :7546 e Vite em :5173 (proxy de /api para a API)
dev-api:
	cd backend && go run ./cmd/server

dev-web:
	cd frontend && npm run dev
