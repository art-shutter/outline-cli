package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/alexflint/go-arg"
	"github.com/art-shutter/outline-cli/internal/api"
	"github.com/art-shutter/outline-cli/internal/config"
	"github.com/dustin/go-humanize"
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
  outline-cli servers add myserver https://example.com/secret --cert-sha256 34B3C8EB1C6EC9B5335556D7E8DC73A30152D27C66B054BAB8ACF5D11AE0C810
  outline-cli servers add-json myserver '{"apiUrl":"https://example.com/secret","certSha256":"34B3C8EB1C6EC9B5335556D7E8DC73A30152D27C66B054BAB8ACF5D11AE0C810"}'
  outline-cli keys create myserver -k "My Key" -l 1GB
  outline-cli keys list myserver
  outline-cli servers metrics myserver

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

type ListKeysCmd struct {
	ServerName string `arg:"positional,required" help:"Server name"`
}

type CreateKeyCmd struct {
	ServerName string           `arg:"positional,required" help:"Server name"`
	Name       string           `arg:"-k,--key-name" help:"Access key name"`
	Method     EncryptionMethod `arg:"-m,--method" default:"aes-192-gcm" help:"Encryption method"`
	Port       Port             `arg:"-p,--port" help:"Port number"`
	DataLimit  DataSize         `arg:"-l,--data-limit" help:"Data limit (e.g., '1GB', '500MB', '2TB')"`
}

type DeleteKeyCmd struct {
	ServerName string `arg:"positional,required" help:"Server name"`
	KeyID      string `arg:"-k,--key-id" help:"Access key ID (use this to delete by ID)"`
	KeyName    string `arg:"-n,--key-name" help:"Access key name (use this to delete by name)"`
}

type EditKeyCmd struct {
	ServerName  string   `arg:"positional,required" help:"Server name"`
	KeyID       string   `arg:"-k,--key-id" help:"Access key ID (use this to edit by ID)"`
	KeyName     string   `arg:"-n,--key-name" help:"Access key name (use this to edit by name)"`
	NewName     string   `arg:"--new-name" help:"New name for the access key"`
	DataLimit   DataSize `arg:"-l,--data-limit" help:"New data limit (e.g., '1GB', '500MB', '2TB')"`
	RemoveLimit bool     `arg:"--remove-limit" help:"Remove data limit from the key"`
}

type MetricsCmd struct {
	ServerName string `arg:"positional,required" help:"Server name"`
}

func main() {
	var args Args
	parser := arg.MustParse(&args)

	slog.SetDefault(newLogger(args.Verbosity))

	if err := validateArgs(&args); err != nil {
		parser.Fail(err.Error())
	}

	if args.Version == nil && args.Servers == nil && args.Keys == nil && args.PrintConfig == nil {
		parser.WriteHelp(os.Stdout)
		return
	}

	if err := run(context.Background(), &args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(parentCtx context.Context, args *Args, out io.Writer) error {
	ctx, stop := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if args.Version != nil {
		_, err := fmt.Fprintf(out, "outline-cli version %s\n", Version)
		return err
	}

	store, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	return execute(ctx, args, store, out)
}

func execute(ctx context.Context, args *Args, store *config.Store, out io.Writer) error {
	switch {
	case args.Servers != nil:
		return handleServersCommand(ctx, args.Servers, store, out)
	case args.Keys != nil:
		return handleKeysCommand(ctx, args.Keys, store, out)
	case args.PrintConfig != nil:
		data, err := store.MarshalYAML()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, string(data))
		return err
	default:
		return nil
	}
}

func handleServersCommand(ctx context.Context, cmd *ServersCmd, store *config.Store, out io.Writer) error {
	switch {
	case cmd.List != nil:
		return listServers(store, out)
	case cmd.Add != nil:
		if err := store.Add(cmd.Add.Name, string(cmd.Add.URL), string(cmd.Add.CertSha256)); err != nil {
			return err
		}
		slog.Info("server added successfully", slog.String("name", cmd.Add.Name))
		return nil
	case cmd.AddJSON != nil:
		if err := store.AddFromJSON(cmd.AddJSON.Name, cmd.AddJSON.JSON); err != nil {
			return err
		}
		slog.Info("server added successfully", slog.String("name", cmd.AddJSON.Name))
		return nil
	case cmd.Get != nil:
		return getServer(ctx, store, cmd.Get.Name, out)
	case cmd.Update != nil:
		return store.UpdateURL(cmd.Update.Name, string(cmd.Update.URL))
	case cmd.Delete != nil:
		return store.Delete(cmd.Delete.Name)
	case cmd.Metrics != nil:
		return getMetrics(ctx, store, cmd.Metrics.ServerName, out)
	default:
		return errors.New("no servers subcommand specified")
	}
}

func handleKeysCommand(ctx context.Context, cmd *KeysCmd, store *config.Store, out io.Writer) error {
	switch {
	case cmd.List != nil:
		return listAccessKeys(ctx, store, cmd.List.ServerName, out)
	case cmd.Create != nil:
		return createAccessKey(ctx, store, cmd.Create, out)
	case cmd.Delete != nil:
		return deleteAccessKey(ctx, store, cmd.Delete)
	case cmd.Edit != nil:
		return editAccessKey(ctx, store, cmd.Edit, out)
	default:
		return errors.New("no keys subcommand specified")
	}
}

func listServers(store *config.Store, out io.Writer) error {
	if len(store.Servers) == 0 {
		slog.Debug("no servers configured")
		return nil
	}

	_, _ = fmt.Fprintln(out, "Configured servers:")
	_, _ = fmt.Fprintln(out, "===================")
	for _, name := range slices.Sorted(maps.Keys(store.Servers)) {
		srv := store.Servers[name]
		_, _ = fmt.Fprintf(out, "Name: %s\n", name)
		_, _ = fmt.Fprintf(out, "URL:  %s\n", srv.URL)
		_, _ = fmt.Fprintf(out, "Cert: %s\n", srv.CertSha256)
		_, _ = fmt.Fprintln(out, "---")
	}
	return nil
}

func getServer(ctx context.Context, store *config.Store, name string, out io.Writer) error {
	srv, err := store.Get(name)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "Server: %s\n", name)
	_, _ = fmt.Fprintf(out, "URL:   %s\n", srv.URL)
	if srv.CertSha256 != "" {
		_, _ = fmt.Fprintf(out, "Cert:  %s\n", srv.CertSha256)
	}

	client := api.NewClient(srv.URL, srv.CertSha256)
	serverInfo, err := client.GetServerInfo(ctx)
	if err != nil {
		slog.Warn("failed to get server info from API", slog.Any("error", err))
		return nil
	}

	_, _ = fmt.Fprintln(out, "API Info:")
	_, _ = fmt.Fprintf(out, "  Name:                    %s\n", serverInfo.Name)
	_, _ = fmt.Fprintf(out, "  Server ID:               %s\n", serverInfo.ServerID)
	_, _ = fmt.Fprintf(out, "  Version:                 %s\n", serverInfo.Version)
	_, _ = fmt.Fprintf(out, "  Metrics Enabled:         %t\n", serverInfo.MetricsEnabled)
	_, _ = fmt.Fprintf(out, "  Port for New Keys:       %d\n", serverInfo.PortForNewAccessKeys)
	_, _ = fmt.Fprintf(out, "  Hostname for Keys:       %s\n", serverInfo.HostnameForAccessKeys)
	if serverInfo.AccessKeyDataLimit != nil {
		_, _ = fmt.Fprintf(out, "  Access Key Data Limit:   %d bytes\n", serverInfo.AccessKeyDataLimit.Bytes)
	}
	return nil
}

func getMetrics(ctx context.Context, store *config.Store, serverName string, out io.Writer) error {
	srv, err := store.Get(serverName)
	if err != nil {
		return err
	}

	client := api.NewClient(srv.URL, srv.CertSha256)
	metrics, err := client.GetTransferMetrics(ctx)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "Transfer metrics for server '%s':\n", serverName)
	_, _ = fmt.Fprintln(out, "==================================")
	if len(metrics.BytesTransferredByUserID) == 0 {
		slog.Debug("no transfer data available", slog.String("server", serverName))
		return nil
	}

	for _, userID := range slices.Sorted(maps.Keys(metrics.BytesTransferredByUserID)) {
		bytes := metrics.BytesTransferredByUserID[userID]
		_, _ = fmt.Fprintf(out, "User %s: %s\n", userID, humanize.Bytes(uint64(max(bytes, 0))))
	}
	return nil
}

func listAccessKeys(ctx context.Context, store *config.Store, serverName string, out io.Writer) error {
	srv, err := store.Get(serverName)
	if err != nil {
		return err
	}

	client := api.NewClient(srv.URL, srv.CertSha256)
	accessKeys, err := client.ListAccessKeys(ctx)
	if err != nil {
		return err
	}

	if len(accessKeys) == 0 {
		slog.Debug("no access keys found on server", slog.String("server", serverName))
		return nil
	}

	_, _ = fmt.Fprintf(out, "Access keys for server '%s':\n", serverName)
	_, _ = fmt.Fprintln(out, "==================================")
	for _, key := range accessKeys {
		_, _ = fmt.Fprintf(out, "ID:       %s\n", key.ID)
		_, _ = fmt.Fprintf(out, "Name:     %s\n", key.Name)
		_, _ = fmt.Fprintf(out, "Port:     %d\n", key.Port)
		_, _ = fmt.Fprintf(out, "Method:   %s\n", key.Method)
		_, _ = fmt.Fprintf(out, "Access URL: %s\n", key.AccessURL)
		if key.DataLimit != nil {
			_, _ = fmt.Fprintf(out, "Data Limit: %s\n", humanize.Bytes(uint64(max(key.DataLimit.Bytes, 0))))
		}
		_, _ = fmt.Fprintln(out, "---")
	}
	return nil
}

func createAccessKey(ctx context.Context, store *config.Store, cmd *CreateKeyCmd, out io.Writer) error {
	srv, err := store.Get(cmd.ServerName)
	if err != nil {
		return err
	}

	req := api.CreateAccessKeyRequest{
		Name:   cmd.Name,
		Method: string(cmd.Method),
		Port:   int(cmd.Port),
	}
	if cmd.DataLimit > 0 {
		req.Limit = &api.DataLimit{Bytes: int64(cmd.DataLimit)}
	}

	client := api.NewClient(srv.URL, srv.CertSha256)
	accessKey, err := client.CreateAccessKey(ctx, req)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out, "Access key created successfully!")
	_, _ = fmt.Fprintf(out, "ID:         %s\n", accessKey.ID)
	_, _ = fmt.Fprintf(out, "Name:       %s\n", accessKey.Name)
	_, _ = fmt.Fprintf(out, "Password:   %s\n", accessKey.Password)
	_, _ = fmt.Fprintf(out, "Port:       %d\n", accessKey.Port)
	_, _ = fmt.Fprintf(out, "Method:     %s\n", accessKey.Method)
	_, _ = fmt.Fprintf(out, "Access URL: %s\n", accessKey.AccessURL)
	if accessKey.DataLimit != nil {
		_, _ = fmt.Fprintf(out, "Data Limit: %s\n", humanize.Bytes(uint64(max(accessKey.DataLimit.Bytes, 0))))
	}
	return nil
}

func deleteAccessKey(ctx context.Context, store *config.Store, cmd *DeleteKeyCmd) error {
	srv, err := store.Get(cmd.ServerName)
	if err != nil {
		return err
	}

	client := api.NewClient(srv.URL, srv.CertSha256)
	keyID, err := resolveKeyID(ctx, client, cmd.ServerName, cmd.KeyID, cmd.KeyName)
	if err != nil {
		return err
	}

	return client.DeleteAccessKey(ctx, keyID)
}

func editAccessKey(ctx context.Context, store *config.Store, cmd *EditKeyCmd, out io.Writer) error {
	srv, err := store.Get(cmd.ServerName)
	if err != nil {
		return err
	}

	client := api.NewClient(srv.URL, srv.CertSha256)
	keyID, err := resolveKeyID(ctx, client, cmd.ServerName, cmd.KeyID, cmd.KeyName)
	if err != nil {
		return err
	}

	if cmd.NewName != "" {
		if err := client.RenameAccessKey(ctx, keyID, cmd.NewName); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Access key renamed successfully to: %s\n", cmd.NewName)
	}

	if cmd.RemoveLimit {
		if err := client.RemoveAccessKeyDataLimit(ctx, keyID); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, "Data limit removed successfully")
	} else if cmd.DataLimit > 0 {
		limit := api.DataLimit{Bytes: int64(cmd.DataLimit)}
		if err := client.SetAccessKeyDataLimit(ctx, keyID, limit); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Data limit updated successfully to: %s\n", cmd.DataLimit.String())
	}

	return nil
}

func resolveKeyID(ctx context.Context, client *api.Client, serverName, keyID, keyName string) (string, error) {
	if keyName != "" {
		keys, err := client.ListAccessKeys(ctx)
		if err != nil {
			return "", err
		}
		for _, k := range keys {
			if k.Name == keyName {
				return k.ID, nil
			}
		}
		return "", fmt.Errorf("access key with name %q not found on server %q", keyName, serverName)
	}
	if keyID == "" {
		return "", errors.New("either --key-id or --key-name must be specified")
	}
	return keyID, nil
}

func newLogger(levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
