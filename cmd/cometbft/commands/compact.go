package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"

	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
)

var CompactCmd = &cobra.Command{
	Use:     "experimental-compact",
	Aliases: []string{"experimental_compact"},
	Short:   "force compacts the CometBFT storage engine (supports GoLevelDB and Pebble)",
	Long: `
This command forces a compaction of the CometBFT storage engine.

Currently, GoLevelDB and Pebble are supported.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}

		conf := cfg.DefaultConfig()
		conf.SetRoot(configFile)

		logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

		switch conf.DBBackend {
		case "goleveldb":
			return compactGoLevelDBs(conf.RootDir, logger)
		case "pebble":
			return compactPebbleDBs(conf.RootDir, logger)
		default:
			return fmt.Errorf("compaction is currently only supported with goleveldb and pebble, got %s", conf.DBBackend)
		}
	},
}

func compactGoLevelDBs(rootDir string, logger log.Logger) error {
	dbNames := []string{"state", "blockstore"}
	o := &opt.Options{
		DisableSeeksCompaction: true,
	}
	wg := sync.WaitGroup{}

	for _, dbName := range dbNames {
		dbName := dbName
		wg.Add(1)
		go func() {
			defer wg.Done()
			dbPath := filepath.Join(rootDir, "data", dbName+".db")
			store, err := leveldb.OpenFile(dbPath, o)
			if err != nil {
				logger.Error("failed to initialize cometbft db", "path", dbPath, "err", err)
				return
			}
			defer store.Close()

			logger.Info("starting compaction...", "db", dbPath)

			err = store.CompactRange(util.Range{Start: nil, Limit: nil})
			if err != nil {
				logger.Error("failed to compact cometbft db", "path", dbPath, "err", err)
			}
		}()
	}
	wg.Wait()
	return nil
}

func compactPebbleDBs(rootDir string, logger log.Logger) error {
	dbNames := []string{"state", "blockstore", "evidence", "tx_index"}
	for _, dbName := range dbNames {
		dbPath := filepath.Join(rootDir, "data", dbName+".db")
		logger.Info("Compacting db", "path", dbPath)

		db, err := pebble.Open(dbPath, &pebble.Options{})
		if err != nil {
			return fmt.Errorf("failed to open db %s: %w", dbPath, err)
		}

		err = db.Compact(nil, nil, true)
		if err != nil {
			db.Close()
			return fmt.Errorf("failed to compact db %s: %w", dbPath, err)
		}

		err = db.Close()
		if err != nil {
			return fmt.Errorf("failed to close db %s: %w", dbPath, err)
		}
	}
	return nil
}
