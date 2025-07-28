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
		Description: "Query information about a package which is installed on the system or available in the repository.",
	}, packageMgr.Query)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_repos",
		Description: "List the configured package repositories on the system, including details such as their names, URLs, and enabled status. This tool provides an overview of where packages are sourced from.",
	}, packageMgr.ListRepo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "modify_repo",
		Description: "Modify a package repository on the system. This can be used to enable, disable, or change the properties of a repository. If the repository does not exist, it will be added. The function can also be used to remove a repository.",
	}, packageMgr.ModifyRepo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_patches",
		Description: "List the available patches on the system, including details such as their names, categories, and severities. This tool provides an overview of the available patches that can be installed.",
	}, packageMgr.ListPatches)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "install_patches",
		Description: "Install patches on the system. This can be used to install all available patches or a subset of patches based on their category or severity.",
	}, packageMgr.InstallPatches)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_package",
		Description: "Search for a package in the enabled repositories. Wildcards are supported.",
	}, packageMgr.SearchPackage)
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
