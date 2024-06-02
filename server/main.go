package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flaviojohansson/goexpert-client-server-api/common"
	"github.com/valyala/fastjson"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /cotacao", cotacaoHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("Server running on port 8080...")

		if err := server.ListenAndServe(); err != nil && http.ErrServerClosed != err {
			log.Fatalf("Could not Listen on %s: %v\n", server.Addr, err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	log.Println("Shutting down server ...")
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Could not gracefully shutdown server: %v\n", err)
	}
	log.Println("Server stopped")

}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {

	// TODO: adicionar middleware para log
	log.Println(r.Method, r.RemoteAddr, r.RequestURI)

	const URL = "https://economia.awesomeapi.com.br/json/last/USD-BRL"

	// TODO: ctx := r.Context()
	// e toda a chamada pra cotacao dentro de go routive.
	// criar channel (bool  mesmo)
	// no fim do request, channel<-true
	// select {
	// case <-chan
	//     log.Println("Request processada com sucesso")
	//     w.Write([]byte("Request processada com sucesso"))
	// case <-ctx.Done():
	//     log.Println("Request cancelada pelo cliente")
	// }

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		log.Panicln(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Request error: %s %v\n", URL, err)

		var statusCode int

		// Existe algum outro tipo de erro aqui que não seja timeout e que eu precise tratar ?
		if ctx.Err().Error() == "context deadline exceeded" {
			statusCode = http.StatusGatewayTimeout
		} else {
			statusCode = http.StatusInternalServerError
		}
		w.WriteHeader(statusCode)
		return
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Printf("Request error: %s %s\n", URL, res.Status)
		// Qualquer erro com a API retornaremos erro 500
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Reading response body error:  %v\n", err)
	}

	var p fastjson.Parser
	v, err := p.Parse(string(body))
	if err != nil {
		log.Printf("Parsing JSON error: %v\n", err)
	}

	cotacaoJSON := v.Get("USDBRL").String()

	var cotacao common.Cotacao
	if err = json.Unmarshal([]byte(cotacaoJSON), &cotacao); err != nil {
		log.Panicf("Parsing JSON error %v\n", err)
	}

	// TODO Gravar em banco de dados
	// contect timeout: 10 ms

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cotacao)

}