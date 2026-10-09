MUNICIPIO ?= LIVRAMENTO DE NOSSA SENHORA
NOME ?= Livramento de Nossa Senhora

.PHONY: dados site serve deploy

# lê o bweb do TSE e gera data/livramento_*.csv
dados:
	go run ./cmd/filtrar -municipio "$(MUNICIPIO)"

# converte os CSVs em site/data.json
site: dados
	go run ./cmd/site -municipio "$(NOME)"

serve:
	cd site && python3 -m http.server 8000

deploy:
	vercel --prod
