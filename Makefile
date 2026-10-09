MUNICIPIO ?= LIVRAMENTO DE NOSSA SENHORA
NOME ?= Livramento de Nossa Senhora

.PHONY: dados site serve deploy pdf

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

# gera o PDF a partir de site/relatorio.html (precisa de `make serve` rodando)
CHROME ?= /mnt/c/Program Files/Google/Chrome/Application/chrome.exe
pdf:
	"$(CHROME)" --headless=new --disable-gpu --no-pdf-header-footer --virtual-time-budget=15000 \
		--print-to-pdf="$$(wslpath -w "$$PWD")\\relatorio-abstencao-livramento.pdf" http://localhost:8000/relatorio.html
