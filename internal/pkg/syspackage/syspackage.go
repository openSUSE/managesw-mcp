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
	ListReposSysCall() (ret []map[string]any, err error)
}

type ListPackageParams struct {
	Name string `json:"name" jsonschema:"Name pattern of the packages to be listed. Using an empty string will result in a list of all packages installed on the system."`
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

func (sysPkg SysPackage) ListRepo(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[any]) (toolRes *mcp.CallToolResultFor[any], err error) {
	result, err := sysPkg.ListReposSysCall()
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
