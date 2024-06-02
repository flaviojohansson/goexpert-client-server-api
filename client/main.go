package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/flaviojohansson/goexpert-client-server-api/common"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	const URL = "http://localhost:8080/cotacao"
	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)

	if err != nil {
		log.Panicln(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Erro na requisição : %v\n", err)
		return
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Printf("Erro na requisição : %s %s\n", URL, res.Status)
		return
	}

	jsonData, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Erro ao ler resposta %v\n", err)
	}

	var cotacao common.Cotacao

	json.Unmarshal(jsonData, &cotacao)

	// Para criar e/ou abrir arquivo para append
	// f, err := os.OpenFile("./cotacao.txt", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)

	f, err := os.Create("./cotacao.txt")

	if err != nil {
		log.Panicln(err)
	}
	defer f.Close()

	fmt.Fprintf(f, "Dólar:%s\n", cotacao.Bid)

	log.Printf("Processo finalizado com sucesso. R$ %s\n", cotacao.Bid)

}
