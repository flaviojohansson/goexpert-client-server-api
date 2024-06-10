package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flaviojohansson/goexpert-client-server-api/common"
	_ "github.com/mattn/go-sqlite3"
	"github.com/valyala/fastjson"
)

var db *sql.DB

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./cotacao.db")
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := "CREATE TABLE IF NOT EXISTS cotacao (dtCotacao datetime, cotacao text)"
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func main() {
	db = initDB()
	defer db.Close()

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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
	const API_DEADLINE = 200 //ms
	const DB_DEADLINE = 10   //ms

	var cotacao common.Cotacao
	mainChan := make(chan bool)
	mainCtx := r.Context()

	go func() {

		ctx, cancel := context.WithTimeout(context.Background(), API_DEADLINE*time.Millisecond)
		defer cancel()

		//
		// Nota de esclarecimento:
		// Se sair da go routine por causa de algum return ou panic,
		// faz com que o processo termine smoothly e não fique parado lá no select{}
		//
		defer func() { mainChan <- false }()

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

		if err = json.Unmarshal([]byte(cotacaoJSON), &cotacao); err != nil {
			log.Panicf("Parsing JSON error %v\n", err)
		}

		DBctx, DBcancel := context.WithTimeout(context.Background(), DB_DEADLINE*time.Millisecond)
		defer DBcancel()
		stmt, err := db.Prepare("INSERT INTO cotacao (dtCotacao, cotacao) VALUES (?, ?)")
		if err != nil {
			log.Panicf("DB Prepare error: %v\n", err)
		}
		defer stmt.Close()
		_, err = stmt.ExecContext(DBctx, time.Now(), cotacao.Bid)
		if err != nil {
			log.Printf("DB Exec error: %v\n", err)
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}

		// chegou aqui, tudo na paz
		mainChan <- true
	}()

	select {
	case ok := <-mainChan:
		if ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cotacao)
		}
	case <-mainCtx.Done():
		log.Println("Connection closed by remote host")
	}

}
