package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/gophertribe/devtool/build"
)

func BuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build sns application",
		RunE: func(cmd *cobra.Command, args []string) error {
			os := cmd.Flag("os").Value.String()
			arch := cmd.Flag("arch").Value.String()
			version := cmd.Flag("version").Value.String()
			crossOs := cmd.Flag("cross-os").Value.String()
			crossArch := cmd.Flag("cross-arch").Value.String()

			// if this is a native build, use go build
			if os == runtime.GOOS && arch == runtime.GOARCH {
				if crossOs != "" && crossArch != "" {
					os = crossOs
					arch = crossArch
				}
				return build.GoBuild("dist/sns", "./cmd/sensors", build.GoBuildOpts{
					Version:       version,
					InjectVersion: true,
					ConfigPackage: "github.com/mklimuk/sensors/pkg/config",
					EnableCgo:     true,
					Arch:          arch, // if cross-arch is not set, it will use runtime.GOARCH
					OS:            os,   // if cross-os is not set, it will use runtime.GOOS
				})
			}

			noCache, err := cmd.Flags().GetBool("no-cache")
			if err != nil {
				return fmt.Errorf("could not get no-cache flag: %w", err)
			}
			return build.Docker(cmd.Context(), fmt.Sprintf("./dev-%s-%s", os, arch), []string{"build", "--version", version, "--cross-os", crossOs, "--cross-arch", crossArch}, build.DockerBuildOpts{
				NoCache: noCache,
				Image:   "gophertribe/gobuild:1.25-bookworm",
			})
		},
	}
	cmd.Flags().Bool("no-cache", false, "do not use cache when building the app")
	cmd.Flags().String("version", "latest", "version of the cli")
	cmd.Flags().String("os", runtime.GOOS, "os to build for")
	cmd.Flags().String("arch", runtime.GOARCH, "arch to build for")
	cmd.Flags().String("cross-os", "", "os to cross-compile for")
	cmd.Flags().String("cross-arch", "", "arch to cross-compile for")

	return cmd
}
