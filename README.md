# map-urnas

Abstenção por bairro, local de votação e seção em Livramento de Nossa Senhora (BA),
a partir dos dados abertos do TSE. Site estático em `site/`, pronto para a Vercel.

## Dados (não versionados — baixe do TSE)

- `data/bweb_1t_BA_*.csv` — Boletim de Urna, 1º turno, BA
  (dadosabertos.tse.jus.br → Resultados 2026 → Boletim de urna)
- `data/locais_tse/eleitorado_local_votacao_2026_BA.csv` — locais de votação
  (https://cdn.tse.jus.br/estatistica/sead/odsele/eleitorado_locais_votacao/eleitorado_local_votacao_2026.zip)

## Gerar

```sh
make site                       # filtra o bweb (~20s) e gera site/data.json
make site MUNICIPIO="BRUMADO" NOME="Brumado"   # outro município da BA
make serve                      # http://localhost:8000
```

Saídas intermediárias em `data/livramento_{bweb,secoes,locais,bairros}.csv`.

## Publicar na Vercel

A Vercel serve `site/` direto, sem build (ver `vercel.json`):

- **CLI:** `npm i -g vercel` e depois `vercel --prod` na raiz do projeto.
- **GitHub:** suba o repositório e importe em vercel.com/new. O `.gitignore` já exclui os CSVs brutos do TSE.
