package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mikoto2000/oasiz_postgres_mcp/internal/config"
	"github.com/mikoto2000/oasiz_postgres_mcp/internal/mcpserver"
	"github.com/mikoto2000/oasiz_postgres_mcp/internal/metadata"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "unknown"

func main() {
	showVersion := flag.Bool("v", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	repo := metadata.NewRepository(pool, cfg.Metadata)
	server := mcpserver.New(repo)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
