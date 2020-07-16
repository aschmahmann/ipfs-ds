package main

import (
	"fmt"
	"os"

	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multibase"

	"github.com/ipfs/go-ipfs/plugin"
	"github.com/ipfs/go-ipfs/plugin/plugins/badgerds"
	"github.com/ipfs/go-ipfs/plugin/plugins/flatfs"
	"github.com/ipfs/go-ipfs/plugin/plugins/levelds"
	"github.com/ipfs/go-ipfs/repo/fsrepo"

	"github.com/urfave/cli/v2"
)

func init() {
	badgerPlugin := badgerds.Plugins[0].(plugin.PluginDatastore)
	flatfsPlugin := flatfs.Plugins[0].(plugin.PluginDatastore)
	leveldsPlugin := levelds.Plugins[0].(plugin.PluginDatastore)
	plugins := []plugin.PluginDatastore{badgerPlugin, flatfsPlugin, leveldsPlugin}

	for _, pl := range plugins {
		if err := fsrepo.AddDatastoreConfigHandler(pl.DatastoreTypeName(), pl.DatastoreConfigParser()); err != nil {
			panic(err)
		}
	}
}

func main() {
	var prefix, numeric bool
	var valueBase string
	var keyMultibase, valueMultibase bool
	var ipfsPath string

	app := &cli.App{
		Name: "ipfs-ds",
		Commands: []*cli.Command{
			{
				Name:    "get",
				Usage:   "get a datastore value by key",
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

					ds, err := GetDatastore(ipfsPath)
					if err != nil {
						return err
					}

					val, err := GetDatastoreValue(ds, c.Args().First(), valueBase, keyMultibase)
					if err != nil {
						return err
					}

					fmt.Println(val)
					return nil
				},
			},
			{
				Name:    "put",
				Usage:   "put a datastore key-value pair",
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

					ds, err := GetDatastore(ipfsPath)
					if err != nil {
						return err
					}

					return SetDatastoreValue(ds, args.First(), args.Get(1), keyMultibase, valueMultibase)
				},
			},
			{
				Name:    "bases",
				Usage:   "List available multibase encodings",
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

func GetDatastore(ipfsPath string) (datastore.Datastore, error) {
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

	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	return repo.Datastore(), nil
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
