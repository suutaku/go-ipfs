package main

import (
	"io"
	"os"

	cmds "github.com/ipfs/go-ipfs-cmds"
	config "github.com/ipfs/go-ipfs-config"
	oldcmds "github.com/ipfs/go-ipfs/commands"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/ipfs/interface-go-ipfs-core/options"
)

var rotateCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Rotates the ipfs identity.",
		ShortDescription: `
Generates a new ipfs identity and saves it to the ipfs config file.
The daemon must not be running when calling this command.

ipfs uses a repository in the local file system. By default, the repo is
located at ~/.ipfs. To change the repo location, set the $IPFS_PATH
environment variable:

    export IPFS_PATH=/path/to/ipfsrepo
`,
	},
	Arguments: []cmds.Argument{},
	Options: []cmds.Option{
		cmds.IntOption(bitsOptionName, "b", "Number of bits to use in the generated RSA private key.").WithDefault(nBitsForKeypairDefault),
	},
	PreRun: func(req *cmds.Request, env cmds.Environment) error {
		cctx := env.(*oldcmds.Context)
		daemonLocked, err := fsrepo.LockedByOtherProcess(cctx.ConfigRoot)
		if err != nil {
			return err
		}

		log.Info("checking if daemon is running...")
		if daemonLocked {
			log.Debug("ipfs daemon is running")
			e := "ipfs daemon is running. please stop it to run this command"
			return cmds.ClientError(e)
		}

		return nil
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		cctx := env.(*oldcmds.Context)
		nBitsForKeypair, _ := req.Options[bitsOptionName].(int)
		return doRotate(os.Stdout, cctx.ConfigRoot, nBitsForKeypair)
	},
}

func doRotate(out io.Writer, repoRoot string, nBitsForKeypair int) error {
	// Open repo
	if err := checkWritable(repoRoot); err != nil {
		return err
	}
	repo, err := fsrepo.Open(repoRoot)
	if err != nil {
		return err
	}
	defer repo.Close()

	// Read config file from repo
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	// Save old identity to keystore
	privKey, err := cfg.Identity.DecodePrivateKey("") //XXX
	if err != nil {
		return err
	}
	keystore := repo.Keystore()
	if err := keystore.Put(cfg.Identity.PeerID, privKey); err != nil {
		return err
	}

	// Generate new identity
	identity, err := config.CreateIdentity(out, []options.KeyGenerateOption{options.Key.Size(nBitsForKeypair)})
	if err != nil {
		return err
	}
	cfg.Identity = identity

	// Write config file to repo
	return repo.SetConfig(cfg)
}
