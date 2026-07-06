package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"

	_ "embed"

	"github.com/cheynewallace/tabby"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/suse/managesw-mcp/internal/pkg/oscheck"
	"github.com/suse/managesw-mcp/internal/pkg/syspackage"
)

//go:embed VERSION
var version string

func NewRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:     "managesw-mcp",
		Short:   "OS software management MCP server",
		Version: strings.TrimSpace(version),
		RunE: func(cmd *cobra.Command, args []string) error {
			viper.SetEnvPrefix("MANAGESW_MCP")
			viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
			viper.AutomaticEnv()
			viper.BindPFlags(cmd.Flags())

			logLevel := slog.LevelInfo
			if viper.GetBool("debug") {
				logLevel = slog.LevelDebug
			}
			handlerOpts := &slog.HandlerOptions{
				Level: logLevel,
			}
			var logger *slog.Logger
			logOutput := os.Stderr
			if viper.GetString("logfile") != "" {
				f, err := os.OpenFile(viper.GetString("logfile"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
				if err != nil {
					return fmt.Errorf("failed to open log file: %w", err)
				}
				defer f.Close()
				logOutput = f
			}

			// Choose handler based on format preference
			if viper.GetBool("log-json") {
				logger = slog.New(slog.NewJSONHandler(logOutput, handlerOpts))
			} else {
				logger = slog.New(slog.NewTextHandler(logOutput, handlerOpts))
			}
			slog.SetDefault(logger)
			slog.Debug("Logger initialized", "level", logLevel)

			server := mcp.NewServer(&mcp.Implementation{
				Name:    "OS software management",
				Version: strings.TrimSpace(version),
			}, nil)

			root := viper.GetString("root")
			packageMgr := oscheck.NewPkg(root)
			listSchema, err := packageMgr.CreateListPackageSchema()
			if err != nil {
				return err
			}
			querySchema, err := syspackage.GetQueryPackageParamsSchema()
			if err != nil {
				return err
			}
			searchSchema, err := packageMgr.CreateSearchPackageSchema()
			if err != nil {
				return err
			}
			installSchema, err := packageMgr.CreateInstallPackageSchema()
			if err != nil {
				return err
			}

			tools := []struct {
				Tool     *mcp.Tool
				Register func(server *mcp.Server, tool *mcp.Tool)
			}{
				{
					Tool: &mcp.Tool{
						Name:        "list_packages",
						Description: "List the installed packages on the system.",
						InputSchema: listSchema,
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.List)
					},
				},
				{
					Tool: &mcp.Tool{
						Name:        "query_package",
						Description: "Query information about a package which is installed on the system or available in the repository.",
						InputSchema: querySchema,
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.Query)
					},
				},
				{
					Tool: &mcp.Tool{
						Name:        "list_repos",
						Description: "List the configured package repositories on the system, including details such as their names, URLs, and enabled status. This tool provides an overview of where packages are sourced from.",
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.ListRepo)
					},
				},
				{
					Tool: &mcp.Tool{
						Name:        "modify_repo",
						Description: "Modify a package repository on the system. This can be used to enable, disable, or change the properties of a repository. If the repository does not exist, it will be added. The function can also be used to remove a repository.",
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.ModifyRepo)
					},
				},
				{
					Tool: &mcp.Tool{
						Name:        "list_patches",
						Description: "List the available patches on the system, including details such as their names, categories, and severities. This tool provides an overview of the available patches that can be installed.",
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.ListPatches)
					},
				},
				{
					Tool: &mcp.Tool{
						Name:        "install_patches",
						Description: "Install patches on the system. This can be used to install all available patches or a subset of patches based on their category or severity.",
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.InstallPatches)
					},
				},
				{
					Tool: &mcp.Tool{
						Name:        "search_package",
						Description: "Search for a package in the enabled repositories. Wildcards are supported.",
						InputSchema: searchSchema,
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.SearchPackage)
					},
				},
				{
					Tool: &mcp.Tool{
						Name:        "install_package",
						Description: "Install a package and its dependencies on the system from the online repositories.",
						InputSchema: installSchema,
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.InstallPackage)
					},
				},
				{
					Tool: &mcp.Tool{
						Name:        "remove_package",
						Description: "Remove a package and its dependencies on the system.",
					},
					Register: func(server *mcp.Server, tool *mcp.Tool) {
						mcp.AddTool(server, tool, packageMgr.RemovePackage)
					},
				},
			}

			var allTools []string
			for _, tool := range tools {
				allTools = append(allTools, tool.Tool.Name)
			}
			if viper.GetBool("list-tools") {
				if viper.GetBool("verbose") {
					tb := tabby.New()
					tb.AddHeader("TOOL", "DESCRIPTION")
					for _, tool := range tools {
						tb.AddLine(tool.Tool.Name, tool.Tool.Description)
					}
					tb.Print()
				} else {
					fmt.Println(strings.Join(allTools, ","))
				}
				return nil
			}

			var enabledTools []string
			if !cmd.Flags().Changed("enabled-tools") {
				enabledTools = allTools
			} else {
				enabledTools = viper.GetStringSlice("enabled-tools")
			}
			// register the enabled tools
			for _, tool := range tools {
				if slices.Contains(enabledTools, tool.Tool.Name) {
					tool.Register(server, tool.Tool)
				}
			}

			if httpAddr := viper.GetString("http"); httpAddr != "" {
				handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
					return server
				}, nil)
				if viper.GetString("cert-file") == "" {
					slog.Info("MCP handler listening at", slog.String("address", httpAddr))
					if err := http.ListenAndServe(httpAddr, handler); err != nil {
						slog.Error("couldn't start http server", "error", err)
						return err
					}
				} else {
					keyFile := viper.GetString("key-file")
					certFile := viper.GetString("cert-file")
					slog.Info("MCP handler listening with TLS at", slog.String("address", httpAddr))
					if err := http.ListenAndServeTLS(httpAddr, certFile, keyFile, handler); err != nil {
						slog.Error("couldn't start tls http server", "error", err)
						return err
					}
				}
			} else {
				slog.Debug("New client has connected via stdin/stdout")
				t := &mcp.LoggingTransport{
					Transport: &mcp.StdioTransport{},
					Writer:    os.Stdout,
				}
				if err := server.Run(context.Background(), t); err != nil {
					slog.Error("Server failed", slog.Any("error", err))
					return err
				}
			}

			return nil
		},
	}

	rootCmd.Flags().String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")
	rootCmd.Flags().String("logfile", "", "if set, log to this file instead of stderr")
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().BoolP("debug", "d", false, "Enable debug logging")
	rootCmd.Flags().Bool("log-json", false, "Output logs in JSON format (machine-readable)")
	rootCmd.Flags().Bool("list-tools", false, "List all available tools and exit")
	rootCmd.Flags().StringSlice("enabled-tools", nil, "A list of tools to enable. Defaults to all tools.")
	rootCmd.Flags().String("cert-file", "", "Path to server certificate file (PEM format) for TLS. Requires --key-file")
	rootCmd.Flags().String("key-file", "", "Path to server private key file (PEM format) for TLS. Requires --cert-file")
	rootCmd.Flags().String("root", "", "if set, use this directory as the root for package operations")

	rootCmd.MarkFlagsRequiredTogether("cert-file", "key-file")

	return rootCmd
}

func main() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
