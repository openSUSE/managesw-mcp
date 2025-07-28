package syspackage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SysPackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"vers"`
	Size    uint64 `json:"size"`
}
type SysPackageInterface interface {
	ListInstalledPackagesSysCall(name string) ([]SysPackageInfo, error)
	QueryPackageSysCall(name string, mode QueryMode, lines int) (ret map[string]any, err error)
	ListReposSysCall(name string) (ret []map[string]any, err error)
	ModifyRepoSysCall(params ModifyRepoParams) (ret map[string]any, err error)
	ListPatchesSysCall(params ListPatchesParams) ([]map[string]any, error)
	InstallPatchesSysCall(params InstallPatchesParams) ([]map[string]any, error)
}

type ListPackageParams struct {
	Name string `json:"name" jsonschema:"Name pattern of the packages to be listed. Using an empty string will result in a list of all packages installed on the system."`
}

type ListPatchesParams struct {
	Category string `json:"category,omitempty" jsonschema:"Category of the patches to be listed."`
	Severity string `json:"severity,omitempty" jsonschema:"Severity of the patches to be listed."`
}

type InstallPatchesParams struct {
	Category string `json:"category,omitempty" jsonschema:"Category of the patches to be installed."`
	Severity string `json:"severity,omitempty" jsonschema:"Severity of the patches to be installed."`
}

type SysPackage struct {
	SysPackageInterface
}

func (sysPkg SysPackage) List(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListPackageParams]) (*mcp.CallToolResultFor[any], error) {
	list, err := sysPkg.ListInstalledPackagesSysCall(params.Arguments.Name)
	if err != nil {
		return nil, err
	}
	txtContentList := []mcp.Content{}
	for _, pkg := range list {
		jsonByte, err := json.Marshal(pkg)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshall packageInfo: %w", err)
		}
		txtContentList = append(txtContentList, &mcp.TextContent{
			Text: string(jsonByte),
		})

	}

	return &mcp.CallToolResultFor[any]{
		Content: txtContentList,
	}, nil

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
	schema, err := jsonschema.For[QueryPackageParams]()
	if err != nil {
		return nil, err
	}
	validList := []any{}
	for _, s := range ValidQueryModes() {
		validList = append(validList, any(s))
	}
	schema.Properties["mode"].Enum = validList
	schema.Properties["mode"].Default = []byte("info")
	return schema, nil
}

func (sysPkg SysPackage) Query(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[QueryPackageParams]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.Name == "" {
		return nil, fmt.Errorf("name for package to query is mandatory")
	}
	mode := getQueryModeFromString(params.Arguments.Mode)
	if mode == -1 {
		return nil, fmt.Errorf("invalid mode: %s valid modes: %v", params.Arguments.Mode, ValidQueryModes())
	}
	result, err := sysPkg.QueryPackageSysCall(params.Arguments.Name, mode, params.Arguments.Lines)
	if err != nil {
		return nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error on qery, couldn't marshall result: %s", result)
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil
}

type ListReposParam struct {
	Name string `json:"name,omitempty" jsonschema:"Name of the repository to list. When omitted all repos are listed."`
}

func (sysPkg SysPackage) ListRepo(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListPackageParams]) (toolRes *mcp.CallToolResultFor[any], err error) {
	result, err := sysPkg.ListReposSysCall(params.Arguments.Name)
	if err != nil {
		return nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error on query, couldn't marshall result: %s", result)
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil
}

type ModifyRepoParams struct {
	Name        string `json:"name" jsonschema:"Name of the rpository"`
	Disable     bool   `json:"disable,omitempty" jsonschema:"Disable the respository"`
	Url         string `json:"url,omitempty" jsonschema:"The uri used for this repository. Use http[s]://url for remote repositories or the full pathname for a local repository."`
	NoGPGCheck  bool   `json:"nogpg,omitempty" jsonschema:"Disable the GPG signature check for the repository"`
	RemoveRepos bool   `json:"removerepo,omitempty" jsonschema:"Remove the repository from the system."`
}

func (sysPkg SysPackage) ModifyRepo(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ModifyRepoParams]) (toolRes *mcp.CallToolResultFor[any], err error) {
	result, err := sysPkg.ModifyRepoSysCall(params.Arguments)
	if err != nil {
		return nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error on query, couldn't marshall result: %s", result)
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil
}

func (sysPkg SysPackage) ListPatches(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListPatchesParams]) (toolRes *mcp.CallToolResultFor[any], err error) {
	result, err := sysPkg.ListPatchesSysCall(params.Arguments)
	if err != nil {
		return nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error on query, couldn't marshall result: %s", result)
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil
}

func (sysPkg SysPackage) InstallPatches(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[InstallPatchesParams]) (toolRes *mcp.CallToolResultFor[any], err error) {
	result, err := sysPkg.InstallPatchesSysCall(params.Arguments)
	if err != nil {
		return nil, err
	}
	jsonByte, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error on query, couldn't marshall result: %s", result)
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonByte),
			},
		},
	}, nil
}
