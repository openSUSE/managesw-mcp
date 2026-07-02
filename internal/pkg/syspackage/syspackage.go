package syspackage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SysPackageInfo struct {
	Name        string              `json:"name"`
	Version     string              `json:"vers"`
	Size        uint64              `json:"size"`
	FileList    []string            `json:"file_list,omitempty"`
	Relations   map[string][]string `json:"relations,omitempty"`
	Description string              `json:"description,omitempty"`
	Changelog   string              `json:"changelog,omitempty"`
}
type SearchedPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}
type SysPackageInterface interface {
	ListInstalledPackagesSysCall(params ListPackageParams) ([]SysPackageInfo, error)
	QueryPackageSysCall(name string, mode QueryMode, lines int) (ret map[string]any, err error)
	ListReposSysCall(name string) (ret []map[string]any, err error)
	RefreshReposSysCall(name string) error
	ModifyRepoSysCall(params ModifyRepoParams) (ret map[string]any, err error)
	ListPatchesSysCall(params ListPatchesParams) ([]map[string]any, error)
	InstallPatchesSysCall(params InstallPatchesParams) ([]map[string]any, error)
	SearchPackageSysCall(params SearchPackageParams) (any, error)
	InstallPackageSysCall(params InstallPackageParams) (string, error)
	RemovePackageSysCall(params RemovePackageParams) (string, error)
	UpdatePackageSysCall(params UpdatePackageParams) (string, error)
	PkgType() string
}

type SysPackage struct {
	SysPackageInterface
}

func (sysPkg SysPackage) List(ctx context.Context, request *mcp.CallToolRequest, params ListPackageParams) (*mcp.CallToolResult, any, error) {
	list, err := sysPkg.ListInstalledPackagesSysCall(params)
	if err != nil {
		return nil, nil, err
	}
	txtContentList := []mcp.Content{}
	for _, pkg := range list {
		jsonByte, err := json.Marshal(pkg)
		if err != nil {
			return nil, nil, fmt.Errorf("could not unmarshal packageInfo: %w", err)
		}
		txtContentList = append(txtContentList, &mcp.TextContent{
			Text: string(jsonByte),
		})

	}

	return &mcp.CallToolResult{
		Content: txtContentList,
	}, nil, nil

}

type QueryMode int

const (
	Info = iota
	Requires
	Recommends
	Obsoletes
	Changelog
)

func getQueryModeFromString(modeStr string) QueryMode {
	switch modeStr {
	case "info":
		return Info
	case "requires":
		return Requires
	case "recommends":
		return Recommends
	case "obsoletes":
		return Obsoletes
	case "changelog":
		return Changelog
	default:
		return -1
	}
}

type QueryPackageParams struct {
	Name  string `json:"name" jsonschema:"Name of the package to be queried."`
	Mode  string `json:"mode" jsonschema:"The mode of the query"`
	Lines int    `json:"lines,omitempty" jsonschema:"The number of lines for 'changelog','recommends','obsoletes','requires'. 'lines' < 0 will show all lines."`
}

func ValidQueryModes() []string {
	return []string{"info", "requires", "recommends", "obsoletes", "changelog"}
}

func GetQueryPackageParamsSchema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[QueryPackageParams](nil)
	if err != nil {
		return nil, err
	}
	validList := []any{}
	for _, s := range ValidQueryModes() {
		validList = append(validList, any(s))
	}
	schema.Properties["mode"].Enum = validList
	schema.Properties["mode"].Default = json.RawMessage("\"info\"")
	return schema, nil
}

func (sysPkg SysPackage) Query(ctx context.Context, request *mcp.CallToolRequest, params QueryPackageParams) (*mcp.CallToolResult, any, error) {
	if params.Name == "" {
		return nil, nil, fmt.Errorf("name for package to query is mandatory")
	}
	mode := getQueryModeFromString(params.Mode)
	if mode == -1 {
		return nil, nil, fmt.Errorf("invalid mode: %s valid modes: %v", params.Mode, ValidQueryModes())
	}
	result, err := sysPkg.QueryPackageSysCall(params.Name, mode, params.Lines)
	if err != nil {
		return nil, nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("error on query, couldn't marshal result: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil, nil
}

type ListPackageParams struct {
	Name        string   `json:"name,omitempty" jsonschema:"Name pattern of the packages to be listed. Using an empty string will result in a list of all packages installed on the system."`
	Filelist    bool     `json:"file_list,omitempty" jsonschema: "List the of the files installed by this package"`
	Relations   []string `json:"relations,omitempty" jsonschema:"Relationship which should be displayed."`
	Description bool     `json:"description,omitempty" jsonschema:"Display also the description of the package"`
	Changelog   uint     `json:"changelog,0" jsonschema:"Show the given number of lines of the changelog."`
}

type ListReposParam struct {
	Name string `json:"name,omitempty" jsonschema:"Name of the repository to list. When omitted all repos are listed."`
}

func (sysPkg SysPackage) ListRepo(ctx context.Context, request *mcp.CallToolRequest, params ListReposParam) (*mcp.CallToolResult, any, error) {
	result, err := sysPkg.ListReposSysCall(params.Name)
	if err != nil {
		return nil, nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("error on query, couldn't marshal result: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil, nil
}

type ModifyRepoParams struct {
	Name        string `json:"name" jsonschema:"Name of the repository"`
	Disable     bool   `json:"disable,omitempty" jsonschema:"Disable the repository"`
	Url         string `json:"url,omitempty" jsonschema:"The URI used for this repository. Use http[s]://url for remote repositories or the full pathname for a local repository."`
	NoGPGCheck  bool   `json:"nogpg,omitempty" jsonschema:"Disable the GPG signature check for the repository"`
	RemoveRepos bool   `json:"removerepo,omitempty" jsonschema:"Remove the repository from the system."`
}

type RefreshReposParams struct {
	Name string `json:"name,omitempty" jsonschema:"Name of the repository to refresh. When omitted all repos are refreshed."`
}

func (sysPkg SysPackage) ModifyRepo(ctx context.Context, request *mcp.CallToolRequest, params ModifyRepoParams) (*mcp.CallToolResult, any, error) {
	result, err := sysPkg.ModifyRepoSysCall(params)
	if err != nil {
		return nil, nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("error on query, couldn't marshal result: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil, nil
}

func (sysPkg SysPackage) RefreshRepos(ctx context.Context, request *mcp.CallToolRequest, params RefreshReposParams) (*mcp.CallToolResult, any, error) {
	err := sysPkg.RefreshReposSysCall(params.Name)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: "Repositories refreshed successfully.",
			},
		},
	}, nil, nil
}

type ListPatchesParams struct {
	Category string `json:"category,omitempty" jsonschema:"Category of the patches to be listed."`
	Severity string `json:"severity,omitempty" jsonschema:"Severity of the patches to be listed."`
}

func (sysPkg SysPackage) ListPatches(ctx context.Context, request *mcp.CallToolRequest, params ListPatchesParams) (*mcp.CallToolResult, any, error) {
	result, err := sysPkg.ListPatchesSysCall(params)
	if err != nil {
		return nil, nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("error on query, couldn't marshal result: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil, nil
}

type InstallPatchesParams struct {
	Category string `json:"category,omitempty" jsonschema:"Category of the patches to be installed."`
	Severity string `json:"severity,omitempty" jsonschema:"Severity of the patches to be installed."`
}

func (sysPkg SysPackage) InstallPatches(ctx context.Context, request *mcp.CallToolRequest, params InstallPatchesParams) (*mcp.CallToolResult, any, error) {
	result, err := sysPkg.InstallPatchesSysCall(params)
	if err != nil {
		return nil, nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("error on query, couldn't marshal result: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil, nil
}

type SearchPackageParams struct {
	Name  string   `json:"name" jsonschema:"Name of the package to search for."`
	Repos []string `json:"repos,omitempty" jsonschema:"A list of repositories to search in. This is optional and should only be used if explicitly requested. If not supplied, all enabled repositories are used."`
	Exact bool     `json:"exact,omitempty" jsonschema:"Match the package name exactly, if not set substrings will also be matched."`
}

func (sysPkg SysPackage) CreateSearchPackageSchema() (*jsonschema.Schema, error) {
	inputSchema, err := jsonschema.For[SearchPackageParams](nil)
	if err != nil {
		return nil, err
	}
	repos, err := sysPkg.ListReposSysCall("")
	if err != nil || len(repos) == 0 {
		return inputSchema, nil
	}

	var validList []any
	for _, repo := range repos {
		var id string
		if v, ok := repo["alias"].(string); ok {
			id = v
		} else if v, ok := repo["Repo-id"].(string); ok {
			id = v
		} else if v, ok := repo["id"].(string); ok {
			id = v
		}
		if id != "" {
			validList = append(validList, id)
		}
	}

	if len(validList) > 0 {
		if inputSchema.Properties["repos"] != nil {
			if inputSchema.Properties["repos"].Items == nil {
				inputSchema.Properties["repos"].Items = &jsonschema.Schema{Type: "string"}
			}
			inputSchema.Properties["repos"].Items.Enum = validList
		}
	}

	return inputSchema, nil
}

func (sysPkg SysPackage) CreateListPackageSchema() (*jsonschema.Schema, error) {
	inputSchema, err := jsonschema.For[ListPackageParams](nil)
	if err != nil {
		return nil, err
	}
	validList := []any{"requires", "recommends", "suggests", "supplements", "enhances", "provides", "conflicts", "obsoletes"}

	if inputSchema.Properties["relations"] != nil {
		if inputSchema.Properties["relations"].Items == nil {
			inputSchema.Properties["relations"].Items = &jsonschema.Schema{Type: "string"}
		}
		inputSchema.Properties["relations"].Items.Enum = validList
	}

	return inputSchema, nil
}

func (sysPkg SysPackage) SearchPackage(ctx context.Context, request *mcp.CallToolRequest, params SearchPackageParams) (*mcp.CallToolResult, any, error) {
	result, err := sysPkg.SysPackageInterface.SearchPackageSysCall(params)
	if err != nil {
		return nil, nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("error on query, couldn't marshal result: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil, nil
}

type InstallPackageParams struct {
	Name            string `json:"name" jsonschema:"Name of the package to install."`
	Version         string `json:"version,omitempty" jsonschema:"Version of the package to install, only needed if alternate version is wanted."`
	FromRepo        string `json:"repo,omitempty" jsonschema:"Repository to install from."`
	WithRecommended bool   `json:"with_recommended,omitempty" jsonschema:"Install recommended packages. These are packages not directly needed, but useful for the package to be functioning."`
	ShowDetails     bool   `json:"show_details,omitempty" jsonschema:"Show which additional packages would be installed, which gives an overview of how much space will consumed."`
}

func (sysPkg SysPackage) InstallPackage(ctx context.Context, request *mcp.CallToolRequest, params InstallPackageParams) (*mcp.CallToolResult, any, error) {
	output, err := sysPkg.SysPackageInterface.InstallPackageSysCall(params)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output,
			},
		},
	}, nil, nil
}

type RemovePackageParams struct {
	Name        string `json:"name" jsonschema:"Name of the package to remove."`
	Purge       bool   `json:"purge,omitempty" jsonschema:"Delete configuration files, etc."`
	RemoveDeps  bool   `json:"removedeps,omitempty" jsonschema:"Automatically remove unneeded dependencies."`
	ShowDetails bool   `json:"show_details,omitempty" jsonschema:"Show which additional packages would be removed."`
}

func (sysPkg SysPackage) RemovePackage(ctx context.Context, request *mcp.CallToolRequest, params RemovePackageParams) (*mcp.CallToolResult, any, error) {
	output, err := sysPkg.SysPackageInterface.RemovePackageSysCall(params)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output,
			},
		},
	}, nil, nil
}

type UpdatePackageParams struct {
	Name    string   `json:"name,omitempty" jsonschema:"Name of the package to update. If omitted, all packages are updated."`
	Repos   []string `json:"repos,omitempty" jsonschema:"A list of repositories to update from."`
	Upgrade bool     `json:"upgrade,omitempty" jsonschema:"On 'zypper', this will perform a 'dup' instead of an 'up'. This has no effect on 'dnf' as it performs an 'upgrade' by default."`
}

func (sysPkg SysPackage) UpdatePackage(ctx context.Context, request *mcp.CallToolRequest, params UpdatePackageParams) (*mcp.CallToolResult, any, error) {
	output, err := sysPkg.SysPackageInterface.UpdatePackageSysCall(params)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output,
			},
		},
	}, nil, nil
}
