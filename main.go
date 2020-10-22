package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multibase"

	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"

	"github.com/urfave/cli/v2"
)

func main() {
	var prefix, numeric bool
	var valueBase string
	var keyMultibase, valueMultibase bool
	var ipfsPath string

	app := &cli.App{
		Name: "ipfs-ds",
		Commands: []*cli.Command{
			{
				Name:  "get",
				Usage: "get a datastore value by key",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Required:    false,
						Name:        "base",
						Aliases:     []string{"b"},
						Value:       "",
						Usage:       "The multibase to encode the value with (e.g. base32)",
						Destination: &valueBase,
					},
					&cli.PathFlag{
						Required:    false,
						Name:        "repo",
						Value:       "",
						Usage:       "The IPFS repo with the datastore config (uses the IPFS_PATH environment variable by default)",
						Destination: &ipfsPath,
					},
					&cli.BoolFlag{
						Required:    false,
						Name:        "key-encoded",
						Value:       false,
						Usage:       "The key is encoded using multibase",
						Destination: &keyMultibase,
					},
				},
				ArgsUsage: "<key>",
				Action: func(c *cli.Context) error {
					if c.NArg() != 1 {
						return fmt.Errorf("incorrect number of arguments")
					}

					repo, err := GetRepo(ipfsPath)
					if err != nil {
						return err
					}
					defer repo.Close()

					val, err := GetDatastoreValue(repo.Datastore(), c.Args().First(), valueBase, keyMultibase)
					if err != nil {
						return err
					}

					fmt.Println(val)
					return nil
				},
			},
			{
				Name:  "put",
				Usage: "put a datastore key-value pair",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Required:    false,
						Name:        "value-encoded",
						Value:       false,
						Usage:       "The multibase to encode the value with (e.g. base32)",
						Destination: &valueMultibase,
					},
					&cli.PathFlag{
						Required:    false,
						Name:        "repo",
						Value:       "",
						Usage:       "The IPFS repo with the datastore config (uses the IPFS_PATH environment variable by default)",
						Destination: &ipfsPath,
					},
					&cli.BoolFlag{
						Required:    false,
						Name:        "key-encoded",
						Value:       false,
						Usage:       "The key is encoded using multibase",
						Destination: &keyMultibase,
					},
				},
				ArgsUsage: "<key> <value>",
				Action: func(c *cli.Context) error {
					args := c.Args()
					if args.Len() != 2 {
						return fmt.Errorf("incorrect number of arguments")
					}

					repo, err := GetRepo(ipfsPath)
					if err != nil {
						return err
					}
					defer repo.Close()

					return SetDatastoreValue(repo.Datastore(), args.First(), args.Get(1), keyMultibase, valueMultibase)
				},
			},
			{
				Name:  "bases",
				Usage: "List available multibase encodings",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Required:    false,
						Name:        "prefix",
						Value:       false,
						Usage:       "also include the single letter prefixes in addition to the code",
						Destination: &prefix,
					},
					&cli.BoolFlag{
						Required:    false,
						Name:        "numeric",
						Value:       false,
						Usage:       "also include numeric codes",
						Destination: &numeric,
					},
				},
				Action: func(c *cli.Context) error {
					printBases(prefix, numeric)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func GetRepo(ipfsPath string) (repo.Repo, error) {
	var repoPath string
	if len(ipfsPath) > 0 {
		repoPath = ipfsPath
	} else {
		var err error
		repoPath, err = fsrepo.BestKnownPath()
		if err != nil {
			return nil, err
		}
	}

	repoLocked, err := fsrepo.LockedByOtherProcess(repoPath)
	if err != nil {
		return nil, err
	}

	if repoLocked {
		return nil, fmt.Errorf("ipfs daemon is running. please stop it to run this command")
	}

	if err := setupPlugins(repoPath); err != nil {
		return nil, err
	}

	return fsrepo.Open(repoPath)
}

func GetDatastoreValue(ds datastore.Datastore, key, outputBase string, keyMultiBase bool) (string, error) {
	var err error

	dsKeyStr := key
	if keyMultiBase {
		_, keyBytes, err := multibase.Decode(key)
		if err != nil {
			return "", err
		}
		dsKeyStr = string(keyBytes)
	}

	dsKey := datastore.NewKey(dsKeyStr)
	val, err := ds.Get(dsKey)
	if err != nil {
		return "", err
	}

	if len(outputBase) == 0 {
		return string(val), nil
	}

	outputBaseEnc, err := BaseEncoderFromString(outputBase)
	if err != nil {
		return "", nil
	}
	return outputBaseEnc.Encode(val), nil
}

func SetDatastoreValue(ds datastore.Datastore, key, value string, keyMultiBase, valueMultiBase bool) error {
	dsKeyStr := key
	if keyMultiBase {
		_, keyBytes, err := multibase.Decode(key)
		if err != nil {
			return err
		}
		dsKeyStr = string(keyBytes)
	}

	dsKey := datastore.NewKey(dsKeyStr)

	dsVal := value
	if valueMultiBase {
		_, valBytes, err := multibase.Decode(value)
		if err != nil {
			return err
		}
		dsVal = string(valBytes)
	}

	return ds.Put(dsKey, []byte(dsVal))
}

func BaseEncoderFromString(base string) (encoder multibase.Encoder, err error) {
	if len(base) == 0 {
		encoder = multibase.MustNewEncoder(multibase.Identity)
	} else {
		encoder, err = multibase.EncoderByName(base)
	}
	return
}
