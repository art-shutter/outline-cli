package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
)

// Version is set during build via ldflags
var Version = "dev"

// VersionCmd represents the version subcommand
type VersionCmd struct{}

// PrintConfigCmd represents the print-config subcommand
type PrintConfigCmd struct{}

// Args represents the command line arguments
type Args struct {
	Version     *VersionCmd     `arg:"subcommand:version" help:"Show version information"`
	Servers     *ServersCmd     `arg:"subcommand:servers" help:"Manage Outline servers"`
	Keys        *KeysCmd        `arg:"subcommand:keys" help:"Manage access keys"`
	PrintConfig *PrintConfigCmd `arg:"subcommand:print-config" help:"Print configuration in YAML format"`
	Verbosity   string          `arg:"-v,--verbosity" default:"info" help:"verbosity level" placeholder:"[error, warning, info, debug]"`
}

// ServersCmd represents the servers subcommand
type ServersCmd struct {
	List    *ListCmd    `arg:"subcommand:list" help:"List all configured servers"`
	Add     *AddCmd     `arg:"subcommand:add" help:"Add a new server"`
	Get     *GetCmd     `arg:"subcommand:get" help:"Get server details"`
	Update  *UpdateCmd  `arg:"subcommand:update" help:"Update server details"`
	Delete  *DeleteCmd  `arg:"subcommand:delete" help:"Delete a server"`
	Metrics *MetricsCmd `arg:"subcommand:metrics" help:"View server metrics"`
}

// ListCmd represents the list subcommand
type ListCmd struct{}

// AddCmd represents the add subcommand
type AddCmd struct {
	Name string `arg:"positional,required" help:"Server name/label"`
	URL  string `arg:"positional,required" help:"Server URL with secret path"`
}

// GetCmd represents the get subcommand
type GetCmd struct {
	Name string `arg:"positional,required" help:"Server name"`
}

// UpdateCmd represents the update subcommand
type UpdateCmd struct {
	Name string `arg:"positional,required" help:"Server name"`
	URL  string `arg:"--url" help:"New server URL"`
}

// DeleteCmd represents the delete subcommand
type DeleteCmd struct {
	Name string `arg:"positional,required" help:"Server name"`
}

// KeysCmd represents the keys subcommand
type KeysCmd struct {
	List   *ListKeysCmd  `arg:"subcommand:list" help:"List access keys"`
	Create *CreateKeyCmd `arg:"subcommand:create" help:"Create a new access key"`
	Delete *DeleteKeyCmd `arg:"subcommand:delete" help:"Delete an access key"`
}

type ServerName struct {
	ServerName string `arg:"-s,--server-name,required" help:"Server name"`
}

// ListKeysCmd represents the list keys subcommand
type ListKeysCmd struct {
	ServerName
}

// CreateKeyCmd represents the create key subcommand
type CreateKeyCmd struct {
	ServerName
	Name   string `arg:"-k,--key-name" help:"Access key name"`
	Method string `arg:"-m,--method" default:"aes-192-gcm" help:"Encryption method"`
	Port   int    `arg:"-p,--port" help:"Port number"`
}

// DeleteKeyCmd represents the delete key subcommand
type DeleteKeyCmd struct {
	ServerName
	KeyID string `arg:"-k,--key-id,required" help:"Access key ID"`
}

// MetricsCmd represents the metrics subcommand
type MetricsCmd struct {
	ServerName
}

func main() {
	var args Args
	parser := arg.MustParse(&args)

	InitLogger(args.Verbosity)

	// Initialize config manager
	configManager, err := NewConfigManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Handle subcommands
	switch {
	case args.Version != nil:
		fmt.Printf("outline-cli version %s\n", Version)
	case args.Servers != nil:
		if err := handleServersCommand(args.Servers, configManager); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case args.Keys != nil:
		if err := handleKeysCommand(args.Keys, configManager); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case args.PrintConfig != nil:
		if err := handlePrintConfigCommand(args.PrintConfig, configManager); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		parser.WriteHelp(os.Stdout)
	}
}

func handleServersCommand(cmd *ServersCmd, configManager *ConfigManager) error {
	switch {
	case cmd.List != nil:
		return configManager.ListServers()
	case cmd.Add != nil:
		return configManager.AddServer(cmd.Add.Name, cmd.Add.URL)
	case cmd.Get != nil:
		return configManager.GetServer(cmd.Get.Name)
	case cmd.Update != nil:
		return configManager.UpdateServer(cmd.Update.Name, cmd.Update.URL)
	case cmd.Delete != nil:
		return configManager.DeleteServer(cmd.Delete.Name)
	case cmd.Metrics != nil:
		return configManager.GetMetrics(cmd.Metrics.ServerName.ServerName)
	default:
		return fmt.Errorf("no subcommand specified")
	}
}

func handleKeysCommand(cmd *KeysCmd, configManager *ConfigManager) error {
	switch {
	case cmd.List != nil:
		return configManager.ListAccessKeys(cmd.List.ServerName.ServerName)
	case cmd.Create != nil:
		return configManager.CreateAccessKey(cmd.Create.ServerName.ServerName, cmd.Create.Name, cmd.Create.Method, cmd.Create.Port)
	case cmd.Delete != nil:
		return configManager.DeleteAccessKey(cmd.Delete.ServerName.ServerName, cmd.Delete.KeyID)
	default:
		return fmt.Errorf("no keys subcommand specified")
	}
}

func handlePrintConfigCommand(_ *PrintConfigCmd, configManager *ConfigManager) error {
	return configManager.PrintConfig()
}
