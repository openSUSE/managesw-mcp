package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suse/managesw-mcp/internal/pkg/oscheck"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

var httpAddr = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
var root = flag.String("root", "", "if set, use this directory as the root for package operations")

func main() {
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "OS software management",
		Version: "0.0.1"}, nil)
	packageMgr := oscheck.NewPkg(*root)
	listSchema, _ := packageMgr.CreateListPackageSchema()
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_packages",
		Description: "List the installed packages on the system.",
		InputSchema: listSchema,
	}, packageMgr.List)

	querySchema, _ := syspackage.GetQueryPackageParamsSchema()
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_package",
		Description: "Query information about a package which is installed on the system or available in the repository.",
		InputSchema: querySchema,
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

	searchSchema, _ := packageMgr.CreateSearchPackageSchema()
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_package",
		Description: "Search for a package in the enabled repositories. Wildcards are supported.",
		InputSchema: searchSchema,
	}, packageMgr.SearchPackage)
	installSchema, _ := packageMgr.CreateInstallPackageSchema()
	mcp.AddTool(server, &mcp.Tool{
		Name:        "install_package",
		Description: "Install a package and its dependencies on the system from the online repositories.",
		InputSchema: installSchema,
	}, packageMgr.InstallPackage)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "remove_package",
		Description: "Remove a package and its dependencies on the system.",
	}, packageMgr.RemovePackage)
	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		slog.Info("MCP handler listening at", slog.String("address", *httpAddr))
		http.ListenAndServe(*httpAddr, handler)
	} else {
		t := &mcp.LoggingTransport{
			Transport: &mcp.StdioTransport{},
			Writer:    os.Stdout,
		}
		if err := server.Run(context.Background(), t); err != nil {
			slog.Error("Server failed", slog.Any("error", err))
		}
	}
}
