package main

import (
	sdk "agones.dev/agones/sdks/go"
	"context"
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

		fmt.Fprintf(writer, "GameServerName: %s", gs.ObjectMeta.Name)
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
