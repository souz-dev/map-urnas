// filtrar lê o boletim de urna (bweb) do TSE em streaming, filtra um
// município e gera:
//   - <prefixo>_bweb.csv   : todas as linhas do município (UTF-8)
//   - <prefixo>_secoes.csv : uma linha por seção (aptos, comparecimento, abstenções)
//   - <prefixo>_locais.csv : agregado por local de votação
//   - <prefixo>_bairros.csv: agregado por bairro/povoado (requer -locais)
//
// Com -locais (eleitorado_local_votacao_AAAA_UF.csv do TSE) as saídas ganham
// nome do local, endereço, bairro, zona urbana/rural e coordenadas.
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// latin1Reader converte ISO-8859-1 para UTF-8 (cada byte é um code point).
type latin1Reader struct {
	r   *bufio.Reader
	buf []byte
}

func (l *latin1Reader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if len(l.buf) > 0 {
			c := copy(p[n:], l.buf)
			l.buf = l.buf[c:]
			n += c
			continue
		}
		b, err := l.r.ReadByte()
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		if b < utf8.RuneSelf {
			p[n] = b
			n++
			continue
		}
		var tmp [2]byte
		utf8.EncodeRune(tmp[:], rune(b))
		l.buf = tmp[:]
	}
	return n, nil
}

type secao struct {
	Zona, Secao, Local     int
	Aptos, Comp, Abst      int
	Agregadas, TipoUrna    string
	BrancosPres, NulosPres int
}

type local struct {
	Zona, Local       int
	Secoes            []int
	Aptos, Comp, Abst int
	Info              localInfo
}

type localInfo struct {
	Nome, Endereco, Bairro, Area, Lat, Lon string
}

type bairro struct {
	Nome, Area        string
	Locais            []string
	Secoes            int
	Aptos, Comp, Abst int
}

// carregaLocais lê o cadastro de locais de votação do TSE e devolve as
// informações por (zona, local), considerando só o 1º turno do município.
func carregaLocais(path, mun string) map[[2]int]localInfo {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(&latin1Reader{r: bufio.NewReaderSize(f, 1<<20)})
	r.Comma = ';'
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		log.Fatal(err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[h] = i
	}
	res := map[[2]int]localInfo{}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if rec[col["NM_MUNICIPIO"]] != mun || rec[col["NR_TURNO"]] != "1" {
			continue
		}
		zona, _ := strconv.Atoi(rec[col["NR_ZONA"]])
		loc, _ := strconv.Atoi(rec[col["NR_LOCAL_VOTACAO"]])
		k := [2]int{zona, loc}
		if _, ok := res[k]; ok {
			continue
		}
		end := strings.TrimSpace(rec[col["DS_ENDERECO"]])
		area := ""
		switch {
		case strings.Contains(end, "ZONA URBANA"):
			area = "URBANA"
		case strings.Contains(end, "ZONA RURAL"):
			area = "RURAL"
		}
		res[k] = localInfo{
			Nome:     strings.TrimSpace(rec[col["NM_LOCAL_VOTACAO"]]),
			Endereco: end,
			Bairro:   strings.TrimSpace(rec[col["NM_BAIRRO"]]),
			Area:     area,
			Lat:      strings.Replace(rec[col["NR_LATITUDE"]], ",", ".", 1),
			Lon:      strings.Replace(rec[col["NR_LONGITUDE"]], ",", ".", 1),
		}
	}
	return res
}

func main() {
	in := flag.String("in", "data/bweb_1t_BA_051020261403.csv", "arquivo bweb do TSE")
	mun := flag.String("municipio", "LIVRAMENTO DE NOSSA SENHORA", "NM_MUNICIPIO a filtrar")
	out := flag.String("out", "data/livramento", "prefixo dos arquivos de saída")
	locPath := flag.String("locais", "data/locais_tse/eleitorado_local_votacao_2026_BA.csv", "cadastro de locais de votação do TSE (vazio para não cruzar)")
	flag.Parse()

	var infos map[[2]int]localInfo
	if *locPath != "" {
		infos = carregaLocais(*locPath, *mun)
	}

	start := time.Now()
	f, err := os.Open(*in)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(&latin1Reader{r: bufio.NewReaderSize(f, 1<<20)})
	r.Comma = ';'
	r.LazyQuotes = true
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		log.Fatal(err)
	}
	header = append([]string(nil), header...)
	col := map[string]int{}
	for i, h := range header {
		col[h] = i
	}
	need := func(name string) int {
		i, ok := col[name]
		if !ok {
			log.Fatalf("coluna %s não encontrada", name)
		}
		return i
	}
	cMun, cZona, cSecao, cLocal := need("NM_MUNICIPIO"), need("NR_ZONA"), need("NR_SECAO"), need("NR_LOCAL_VOTACAO")
	cAptos, cComp, cAbst := need("QT_APTOS"), need("QT_COMPARECIMENTO"), need("QT_ABSTENCOES")
	cAgr, cUrna := need("DS_SECOES_AGREGADAS"), need("DS_TIPO_URNA")
	cCargo, cTipoVot, cVotos := need("DS_CARGO_PERGUNTA"), need("DS_TIPO_VOTAVEL"), need("QT_VOTOS")

	rawF, err := os.Create(*out + "_bweb.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer rawF.Close()
	raw := csv.NewWriter(rawF)
	raw.Comma = ';'
	raw.Write(header)

	secoes := map[[2]int]*secao{}
	total, kept := 0, 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("linha %d: %v", total+2, err)
		}
		total++
		if rec[cMun] != *mun {
			continue
		}
		kept++
		raw.Write(rec)

		atoi := func(i int) int { v, _ := strconv.Atoi(strings.TrimSpace(rec[i])); return v }
		key := [2]int{atoi(cZona), atoi(cSecao)}
		s, ok := secoes[key]
		if !ok {
			// aptos/comparecimento/abstenções se repetem em todas as linhas da seção
			s = &secao{Zona: key[0], Secao: key[1], Local: atoi(cLocal),
				Aptos: atoi(cAptos), Comp: atoi(cComp), Abst: atoi(cAbst),
				Agregadas: rec[cAgr], TipoUrna: rec[cUrna]}
			secoes[key] = s
		}
		if rec[cCargo] == "Presidente" {
			switch rec[cTipoVot] {
			case "Branco":
				s.BrancosPres += atoi(cVotos)
			case "Nulo":
				s.NulosPres += atoi(cVotos)
			}
		}
	}
	raw.Flush()
	if err := raw.Error(); err != nil {
		log.Fatal(err)
	}

	// --- seções ---
	list := make([]*secao, 0, len(secoes))
	for _, s := range secoes {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Local != list[j].Local {
			return list[i].Local < list[j].Local
		}
		return list[i].Secao < list[j].Secao
	})
	writeCSV(*out+"_secoes.csv",
		[]string{"NR_ZONA", "NR_SECAO", "NR_LOCAL_VOTACAO", "NM_LOCAL_VOTACAO", "NM_BAIRRO", "AREA", "QT_APTOS", "QT_COMPARECIMENTO", "QT_ABSTENCOES", "PCT_ABSTENCAO", "BRANCOS_PRESIDENTE", "NULOS_PRESIDENTE", "DS_SECOES_AGREGADAS", "DS_TIPO_URNA"},
		func(w *csv.Writer) {
			for _, s := range list {
				in := infos[[2]int{s.Zona, s.Local}]
				w.Write([]string{itoa(s.Zona), itoa(s.Secao), itoa(s.Local), in.Nome, in.Bairro, in.Area, itoa(s.Aptos), itoa(s.Comp), itoa(s.Abst),
					pct(s.Abst, s.Aptos), itoa(s.BrancosPres), itoa(s.NulosPres), s.Agregadas, s.TipoUrna})
			}
		})

	// --- locais ---
	locais := map[[2]int]*local{}
	var tA, tC, tB int
	for _, s := range list {
		k := [2]int{s.Zona, s.Local}
		l, ok := locais[k]
		if !ok {
			l = &local{Zona: s.Zona, Local: s.Local, Info: infos[k]}
			locais[k] = l
		}
		l.Secoes = append(l.Secoes, s.Secao)
		l.Aptos += s.Aptos
		l.Comp += s.Comp
		l.Abst += s.Abst
		tA, tC, tB = tA+s.Aptos, tC+s.Comp, tB+s.Abst
	}
	ll := make([]*local, 0, len(locais))
	for _, l := range locais {
		ll = append(ll, l)
	}
	sort.Slice(ll, func(i, j int) bool { return ll[i].Abst > ll[j].Abst })
	writeCSV(*out+"_locais.csv",
		[]string{"NR_ZONA", "NR_LOCAL_VOTACAO", "NM_LOCAL_VOTACAO", "NM_BAIRRO", "AREA", "DS_ENDERECO", "LATITUDE", "LONGITUDE", "QT_SECOES", "SECOES", "QT_APTOS", "QT_COMPARECIMENTO", "QT_ABSTENCOES", "PCT_ABSTENCAO"},
		func(w *csv.Writer) {
			for _, l := range ll {
				ss := make([]string, len(l.Secoes))
				for i, v := range l.Secoes {
					ss[i] = itoa(v)
				}
				in := l.Info
				w.Write([]string{itoa(l.Zona), itoa(l.Local), in.Nome, in.Bairro, in.Area, in.Endereco, in.Lat, in.Lon, itoa(len(l.Secoes)), strings.Join(ss, " "),
					itoa(l.Aptos), itoa(l.Comp), itoa(l.Abst), pct(l.Abst, l.Aptos)})
			}
		})

	// --- bairros ---
	semInfo := 0
	if infos != nil {
		bairros := map[string]*bairro{}
		for _, l := range ll {
			nome := l.Info.Bairro
			if nome == "" {
				semInfo++
				nome = "(SEM CADASTRO) local " + itoa(l.Local)
			}
			b, ok := bairros[nome]
			if !ok {
				b = &bairro{Nome: nome, Area: l.Info.Area}
				bairros[nome] = b
			} else if b.Area != l.Info.Area {
				b.Area = "MISTA"
			}
			b.Locais = append(b.Locais, l.Info.Nome)
			b.Secoes += len(l.Secoes)
			b.Aptos += l.Aptos
			b.Comp += l.Comp
			b.Abst += l.Abst
		}
		bl := make([]*bairro, 0, len(bairros))
		for _, b := range bairros {
			bl = append(bl, b)
		}
		sort.Slice(bl, func(i, j int) bool { return bl[i].Abst > bl[j].Abst })
		writeCSV(*out+"_bairros.csv",
			[]string{"NM_BAIRRO", "AREA", "QT_LOCAIS", "LOCAIS", "QT_SECOES", "QT_APTOS", "QT_COMPARECIMENTO", "QT_ABSTENCOES", "PCT_ABSTENCAO", "PCT_DO_TOTAL_ABSTENCOES"},
			func(w *csv.Writer) {
				for _, b := range bl {
					w.Write([]string{b.Nome, b.Area, itoa(len(b.Locais)), strings.Join(b.Locais, " | "), itoa(b.Secoes),
						itoa(b.Aptos), itoa(b.Comp), itoa(b.Abst), pct(b.Abst, b.Aptos), pct(b.Abst, tB)})
				}
			})
		fmt.Printf("bairros/povoados: %d | locais sem cadastro: %d\n", len(bl), semInfo)
	}

	fmt.Printf("linhas lidas: %d | linhas do município: %d | tempo: %s\n", total, kept, time.Since(start).Round(time.Millisecond))
	fmt.Printf("seções: %d | locais: %d\n", len(list), len(ll))
	fmt.Printf("aptos: %d | comparecimento: %d | abstenções: %d (%s%%)\n", tA, tC, tB, pct(tB, tA))
}

func writeCSV(path string, header []string, fill func(*csv.Writer)) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Comma = ';'
	w.Write(header)
	fill(w)
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}

func itoa(v int) string { return strconv.Itoa(v) }

func pct(a, b int) string {
	if b == 0 {
		return "0"
	}
	return strconv.FormatFloat(float64(a)*100/float64(b), 'f', 2, 64)
}
