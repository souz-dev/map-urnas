// site converte os CSVs gerados por filtrar em site/data.json, consumido
// pela página estática em site/index.html.
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	prefix := flag.String("in", "data/livramento", "prefixo dos CSVs gerados por filtrar")
	out := flag.String("out", "site/data.json", "arquivo JSON de saída")
	mun := flag.String("municipio", "Livramento de Nossa Senhora", "nome exibido no site")
	eleicao := flag.String("eleicao", "Eleições Gerais 2026 · 1º turno (04/10/2026)", "descrição da eleição")
	flag.Parse()

	data := map[string]any{
		"municipio": *mun,
		"eleicao":   *eleicao,
		"gerado_em": time.Now().Format("02/01/2006 15:04"),
		"bairros":   readCSV(*prefix + "_bairros.csv"),
		"locais":    readCSV(*prefix + "_locais.csv"),
		"secoes":    readCSV(*prefix + "_secoes.csv"),
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(data); err != nil {
		log.Fatal(err)
	}
	fmt.Println("gerado", *out)
}

// readCSV devolve as linhas como objetos; colunas numéricas viram números.
func readCSV(path string) []map[string]any {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	header := rows[0]
	res := make([]map[string]any, 0, len(rows)-1)
	for _, row := range rows[1:] {
		obj := map[string]any{}
		for i, h := range header {
			v := row[i]
			if n, err := strconv.ParseFloat(v, 64); err == nil && h != "SECOES" && h != "DS_SECOES_AGREGADAS" {
				obj[h] = n
			} else {
				obj[h] = v
			}
		}
		res = append(res, obj)
	}
	return res
}
