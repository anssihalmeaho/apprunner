package main

import (
	"apprunner/codeserver"
	"apprunner/executor"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	serviceName := os.Args[0]

	portPtr := flag.String("port", "8080", "Port number for input")
	codeserverAddrPtr := flag.String("csaddr", "", "address of code server, default is built-in code server")
	packFilenamePtr := flag.String("file", "packs.db", "Filename for package storage")
	flag.Parse()

	store := codeserver.NewBoltStore(*packFilenamePtr)
	if err := store.Open(); err != nil {
		log.Fatalf("Not able to open storage: %v", err)
	}
	defer store.Close()

	cs := codeserver.NewCodeServer(store)
	handlerCol, handlerRes := cs.GetHandler()
	packGetter := func(name string) ([]byte, bool) {
		return store.GetByName(name)
	}
	exeHandlerCol, exeHandlerRes := executor.GetHandler(*codeserverAddrPtr, packGetter)

	mux := http.NewServeMux()
	mux.HandleFunc("/packs", handlerCol)
	mux.HandleFunc("/packs/", handlerRes)
	mux.HandleFunc("/app", exeHandlerCol)
	mux.HandleFunc("/app/", exeHandlerRes)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", *portPtr),
		Handler: mux,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("%s shutdown: %v", serviceName, err)
		}
		close(idleConnsClosed)
	}()

	log.Println(serviceName + " started")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error from ListenAndServe: %v", err)
	}

	<-idleConnsClosed
	log.Println(serviceName + " exit")
}
