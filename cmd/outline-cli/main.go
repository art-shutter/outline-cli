package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/art-shutter/outline-cli/internal/config"
)

var Version = "dev"

type VersionCmd struct{}

type PrintConfigCmd struct{}

type Args struct {
	Version     *VersionCmd     `arg:"subcommand:version" help:"Show version information"`
	Servers     *ServersCmd     `arg:"subcommand:servers" help:"Manage Outline servers"`
	Keys        *KeysCmd        `arg:"subcommand:keys" help:"Manage access keys"`
	PrintConfig *PrintConfigCmd `arg:"subcommand:print-config" help:"Print configuration in YAML format"`
	Verbosity   string          `arg:"-v,--verbosity" default:"info" help:"verbosity level" placeholder:"[error, warning, info, debug]"`
}

func (Args) Description() string {
	return "Outline CLI - A command-line interface for managing Outline VPN servers and access keys"
}

func (Args) Epilogue() string {
	return `Examples:
  outline-cli servers add myserver https://example.com/secret abc123def456...
  outline-cli servers add-json myserver '{"apiUrl":"https://example.com/secret","certSha256":"abc123def456..."}'
  outline-cli keys create -s myserver -k mykey -l 1GB
  outline-cli keys list -s myserver
  outline-cli servers metrics -s myserver

For more information, visit: https://github.com/art-shutter/outline-cli`
}

type ServersCmd struct {
	List    *ListCmd    `arg:"subcommand:list" help:"List all configured servers"`
	Add     *AddCmd     `arg:"subcommand:add" help:"Add a new server with individual parameters"`
	AddJSON *AddJSONCmd `arg:"subcommand:add-json" help:"Add a new server from JSON input"`
	Get     *GetCmd     `arg:"subcommand:get" help:"Get server details"`
	Update  *UpdateCmd  `arg:"subcommand:update" help:"Update server details"`
	Delete  *DeleteCmd  `arg:"subcommand:delete" help:"Delete a server"`
	Metrics *MetricsCmd `arg:"subcommand:metrics" help:"View server metrics"`
}

type ListCmd struct{}

type AddCmd struct {
	Name       string     `arg:"positional,required" help:"Server name/label"`
	URL        ServerURL  `arg:"positional,required" help:"Server URL with secret path"`
	CertSha256 CertSHA256 `arg:"--cert-sha256,required" help:"Certificate SHA256 hash"`
}

type AddJSONCmd struct {
	Name string `arg:"positional,required" help:"Server name/label"`
	JSON string `arg:"positional,required" help:"JSON input with apiUrl and certSha256 fields"`
}

type GetCmd struct {
	Name string `arg:"positional,required" help:"Server name"`
}

type UpdateCmd struct {
	Name string    `arg:"positional,required" help:"Server name"`
	URL  ServerURL `arg:"--url" help:"New server URL"`
}

type DeleteCmd struct {
	Name string `arg:"positional,required" help:"Server name"`
}

type KeysCmd struct {
	List   *ListKeysCmd  `arg:"subcommand:list" help:"List access keys"`
	Create *CreateKeyCmd `arg:"subcommand:create" help:"Create a new access key"`
	Delete *DeleteKeyCmd `arg:"subcommand:delete" help:"Delete an access key"`
	Edit   *EditKeyCmd   `arg:"subcommand:edit" help:"Edit an existing access key"`
}

type ServerName struct {
	ServerName string `arg:"-s,--server-name,required" help:"Server name"`
}

type ListKeysCmd struct {
	ServerName
}

type CreateKeyCmd struct {
	ServerName
	Name      string           `arg:"-k,--key-name" help:"Access key name"`
	Method    EncryptionMethod `arg:"-m,--method" default:"aes-192-gcm" help:"Encryption method"`
	Port      Port             `arg:"-p,--port" help:"Port number"`
	DataLimit DataSize         `arg:"-l,--data-limit" help:"Data limit (e.g., '1GB', '500MB', '2TB')"`
}

type DeleteKeyCmd struct {
	ServerName
	KeyID   string `arg:"-k,--key-id" help:"Access key ID (use this to delete by ID)"`
	KeyName string `arg:"-n,--key-name" help:"Access key name (use this to delete by name)"`
}

type EditKeyCmd struct {
	ServerName
	KeyID       string   `arg:"-k,--key-id" help:"Access key ID (use this to edit by ID)"`
	KeyName     string   `arg:"-n,--key-name" help:"Access key name (use this to edit by name)"`
	NewName     string   `arg:"--new-name" help:"New name for the access key"`
	DataLimit   DataSize `arg:"-l,--data-limit" help:"New data limit (e.g., '1GB', '500MB', '2TB')"`
	RemoveLimit bool     `arg:"--remove-limit" help:"Remove data limit from the key"`
}

type MetricsCmd struct {
	ServerName
}

func main() {
	var args Args
	parser := arg.MustParse(&args)

	config.InitLogger(args.Verbosity)

	if err := validateArgs(&args); err != nil {
		parser.Fail(err.Error())
	}

	configManager, err := config.NewConfigManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

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
		if err := configManager.PrintConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		parser.WriteHelp(os.Stdout)
	}
}

func handleServersCommand(cmd *ServersCmd, configManager *config.ConfigManager) error {
	switch {
	case cmd.List != nil:
		return configManager.ListServers()
	case cmd.Add != nil:
		return configManager.AddServer(cmd.Add.Name, cmd.Add.URL.URL, cmd.Add.CertSha256.Hash)
	case cmd.AddJSON != nil:
		return configManager.AddServerFromJSON(cmd.AddJSON.Name, cmd.AddJSON.JSON)
	case cmd.Get != nil:
		return configManager.GetServer(cmd.Get.Name)
	case cmd.Update != nil:
		return configManager.UpdateServer(cmd.Update.Name, cmd.Update.URL.URL)
	case cmd.Delete != nil:
		return configManager.DeleteServer(cmd.Delete.Name)
	case cmd.Metrics != nil:
		return configManager.GetMetrics(cmd.Metrics.ServerName.ServerName)
	default:
		return fmt.Errorf("no subcommand specified")
	}
}

func handleKeysCommand(cmd *KeysCmd, configManager *config.ConfigManager) error {
	switch {
	case cmd.List != nil:
		return configManager.ListAccessKeys(cmd.List.ServerName.ServerName)
	case cmd.Create != nil:
		return configManager.CreateAccessKey(cmd.Create.ServerName.ServerName, cmd.Create.Name, cmd.Create.Method.Method, cmd.Create.Port.Number, cmd.Create.DataLimit.String())
	case cmd.Delete != nil:
		if cmd.Delete.KeyName != "" {
			return configManager.DeleteAccessKeyByName(cmd.Delete.ServerName.ServerName, cmd.Delete.KeyName)
		}
		return configManager.DeleteAccessKey(cmd.Delete.ServerName.ServerName, cmd.Delete.KeyID)
	case cmd.Edit != nil:
		return configManager.EditAccessKey(cmd.Edit.ServerName.ServerName, cmd.Edit.KeyID, cmd.Edit.KeyName, cmd.Edit.NewName, cmd.Edit.DataLimit.String(), cmd.Edit.RemoveLimit)
	default:
		return fmt.Errorf("no keys subcommand specified")
	}
}
