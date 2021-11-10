package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/superfly/flyctl/cmdctx"
	"github.com/superfly/flyctl/docstrings"
	"github.com/superfly/flyctl/internal/client"
	"github.com/superfly/flyctl/recipes"
)

const (
	MACHINE_ROLE = ".flyd/bin/role"
	MACHINE_
)

func newRecipesCommand(client *client.Client) *Command {
	keystrings := docstrings.Get("recipes")
	cmd := BuildCommandCobra(nil, nil, &cobra.Command{
		Use:   keystrings.Usage,
		Short: keystrings.Short,
		Long:  keystrings.Long,
	}, client)

	newPostgresProvisionRecipeCommand(cmd, client)
	newRollingRebootRecipeCommand(cmd, client)
	newRollinUpgradeRecipeCommand(cmd, client)

	return cmd
}

func newPostgresProvisionRecipeCommand(parent *Command, client *client.Client) *Command {
	keystrings := docstrings.Get("recipes.provision-postgres")
	cmd := BuildCommandKS(parent, runProvisionPostgresRecipe, keystrings, client, requireSession)
	cmd.AddStringFlag(StringFlagOpts{Name: "name", Description: "the name of the new app"})
	cmd.AddIntFlag(IntFlagOpts{Name: "count", Description: "the total number of in-region Postgres machines", Default: 2})
	cmd.AddStringFlag(StringFlagOpts{Name: "region", Description: "the region to launch the new app in"})
	cmd.AddStringFlag(StringFlagOpts{Name: "volume-size", Description: "the size in GB for volumes"})
	cmd.AddStringFlag(StringFlagOpts{Name: "image-ref", Description: "the target image", Default: "flyio/postgres:14"})
	cmd.AddStringFlag(StringFlagOpts{Name: "password", Description: "the default password for the postgres use"})
	cmd.AddStringFlag(StringFlagOpts{Name: "consul-url", Description: "Opt into using an existing consul as the backend store by specifying the target consul url."})
	cmd.AddStringFlag(StringFlagOpts{Name: "etcd-url", Description: "Opt into using an existing etcd as the backend store by specifying the target etcd url."})

	return cmd
}

func newRollingRebootRecipeCommand(parent *Command, client *client.Client) *Command {
	keystrings := docstrings.Get("recipes.rolling-reboot")
	cmd := BuildCommandKS(parent, runRollingRebootRecipe, keystrings, client, requireSession, requireAppName)

	return cmd
}

func newRollinUpgradeRecipeCommand(parent *Command, client *client.Client) *Command {
	keystrings := docstrings.Get("recipes.rolling-upgrade")
	cmd := BuildCommandKS(parent, runRollingUpgradeRecipe, keystrings, client, requireSession, requireAppName)
	cmd.AddStringFlag(StringFlagOpts{Name: "image-ref", Description: "the target image", Default: "flyio/postgres:14"})

	return cmd
}

func runRollingUpgradeRecipe(cmdCtx *cmdctx.CmdContext) error {
	ctx := cmdCtx.Command.Context()
	client := cmdCtx.Client.API()

	app, err := client.GetApp(ctx, cmdCtx.AppName)
	if err != nil {
		return fmt.Errorf("get app: %w", err)
	}

	imageRef := cmdCtx.Config.GetString("image-ref")
	if imageRef == "" {
		return fmt.Errorf("Please specify the target image")
	}

	return recipes.PostgresUpgradeRecipe(ctx, app, imageRef)
}

func runRollingRebootRecipe(cmdCtx *cmdctx.CmdContext) error {
	ctx := cmdCtx.Command.Context()
	client := cmdCtx.Client.API()

	app, err := client.GetApp(ctx, cmdCtx.AppName)
	if err != nil {
		return fmt.Errorf("get app: %w", err)
	}

	return recipes.PostgresRebootRecipe(ctx, app)
}

func runProvisionPostgresRecipe(cmdCtx *cmdctx.CmdContext) error {
	ctx := cmdCtx.Command.Context()
	appName := cmdCtx.Config.GetString("name")
	if appName == "" {
		n, err := inputAppName("", false)
		if err != nil {
			return err
		}
		appName = n
	}

	orgSlug := cmdCtx.Config.GetString("organization")
	org, err := selectOrganization(ctx, cmdCtx.Client.API(), orgSlug, nil)
	if err != nil {
		return err
	}

	regionCode := cmdCtx.Config.GetString("region")
	region, err := selectRegion(ctx, cmdCtx.Client.API(), regionCode)
	if err != nil {
		return err
	}

	consulUrl := cmdCtx.Config.GetString("consul-url")
	etcdUrl := cmdCtx.Config.GetString("etcd-url")

	if consulUrl != "" && etcdUrl != "" {
		return fmt.Errorf("consulUrl and etcdUrl may not both be specified.")
	}

	volumeSize := cmdCtx.Config.GetInt("volume-size")
	if volumeSize == 0 {
		s, err := volumeSizeInput(10)
		if err != nil {
			return err
		}
		volumeSize = s
	}

	count := cmdCtx.Config.GetInt("count")
	password := cmdCtx.Config.GetString("password")
	imageRef := cmdCtx.Config.GetString("image-ref")

	p := recipes.NewPostgresProvisionRecipe(cmdCtx, recipes.PostgresProvisionConfig{
		AppName:      appName,
		Count:        count,
		ImageRef:     imageRef,
		Organization: org,
		Password:     password,
		Region:       region.Code,
		VolumeSize:   volumeSize,
		ConsulUrl:    consulUrl,
		EtcdUrl:      etcdUrl,
	})

	return p.Start()
}