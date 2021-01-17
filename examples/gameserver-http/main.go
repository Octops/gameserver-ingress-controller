package main

import (
	sdk "agones.dev/agones/sdks/go"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	port := flag.String("port", "8088", "The port to listen to http traffic on")
	flag.Parse()

	ctx, _ := context.WithCancel(context.Background())

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	addr := fmt.Sprintf(":%s", *port)
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		gs, err := s.GameServer()
		if err != nil {
			fmt.Fprint(writer, err.Error())
		}

		response := struct {
			Name    string
			Address string
			Status  interface{}
		}{
			Name:    gs.ObjectMeta.Name,
			Address: fmt.Sprintf("%s:%d", gs.Status.Address, gs.Status.Ports[0].Port),
			Status:  gs.Status,
		}

		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(response)
	})

	go doHealth(s, ctx)

	log.Print("Marking this server as ready")
	if err := s.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
	}

	go func() {
		log.Println("Server listening on", addr)
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

	server.Shutdown(context.Background())
}

func doHealth(sdk *sdk.SDK, ctx context.Context) {
	tick := time.Tick(2 * time.Second)
	for {
		if err := sdk.Health(); err != nil {
			log.Fatalf("Could not send health ping: %v", err)
		}
		select {
		case <-ctx.Done():
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
