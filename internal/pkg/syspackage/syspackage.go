package syspackage

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"context"
)

type SysPackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"vers"`
	Size    uint64 `json:"size"`
}
type SysPackageInterface interface {
	ListInstalledPackagesSysCall(name string) ([]SysPackageInfo, error)
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
