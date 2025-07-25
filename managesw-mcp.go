package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suse/managesw-mcp/internal/pkg/oscheck"
)

var httpAddr = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")

func main() {
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "OS software management",
		Version: "0.0.1"}, nil)
	packageMgr := oscheck.NewPkg()
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_packages",
		Description: "List the installed packages on the system.",
	}, packageMgr.List)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_package",
		Description: "Query information about a package.",
	}, packageMgr.Query)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_repos",
		Description: "List the configured package repositories on the system, including details such as their names, URLs, and enabled status. This tool provides an overview of where packages are sourced from.",
	}, packageMgr.ListRepo)
	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		slog.Info("MCP handler listening at", slog.String("address", *httpAddr))
		http.ListenAndServe(*httpAddr, handler)
	} else {
		t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stdout)
		if err := server.Run(context.Background(), t); err != nil {
			slog.Error("Server failed", slog.Any("error", err))
		}
	}
}
