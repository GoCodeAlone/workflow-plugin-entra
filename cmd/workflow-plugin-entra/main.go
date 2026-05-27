package main

import (
	"github.com/GoCodeAlone/workflow-plugin-entra/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewEntraPlugin(), sdk.WithBuildVersion(sdk.ResolveBuildVersion(internal.Version)))
}
