package main

import (
	"fmt"
	"io"
	"os"
	"time"

	cmds "github.com/ipfs/go-ipfs-cmds"
	config "github.com/ipfs/go-ipfs-config"
	oldcmds "github.com/ipfs/go-ipfs/commands"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/ipfs/interface-go-ipfs-core/options"
)

const (
	algorithmDefault    = options.Ed25519Key
	algorithmOptionName = "algorithm"
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
		cmds.StringOption(algorithmOptionName, "a", "Cryptographic algorithm to use for key generation.").WithDefault(algorithmDefault),
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
		algorithm, _ := req.Options[algorithmOptionName].(string)
		return doRotate(os.Stdout, cctx.ConfigRoot, algorithm, nBitsForKeypair)
	},
}

func doRotate(out io.Writer, repoRoot string, algorithm string, nBitsForKeypair int) error {
	// Open repo
	repo, err := fsrepo.Open(repoRoot)
	if err != nil {
		return fmt.Errorf("opening repo (%v)", err)
	}
	defer repo.Close()

	// Read config file from repo
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("reading config from repo (%v)", err)
	}

	// Generate new identity
	identity, err := config.CreateIdentity(out, []options.KeyGenerateOption{
		options.Key.Size(nBitsForKeypair),
		options.Key.Type(algorithm),
	})
	if err != nil {
		return fmt.Errorf("creating identity (%v)", err)
	}

	// Save old identity to keystore
	oldPrivKey, err := cfg.Identity.DecodePrivateKey("") //XXX
	if err != nil {
		return fmt.Errorf("decoding old private key (%v)", err)
	}
	keystore := repo.Keystore()
	name := fmt.Sprintf("rotation %s -> %s on %s",
		shorten(cfg.Identity.PrivKey),
		shorten(identity.PrivKey),
		time.Now().Format(time.RFC822),
	)
	if err := keystore.Put(name, oldPrivKey); err != nil {
		return fmt.Errorf("saving old key in keystore (%v)", err)
	}

	// Update identity
	cfg.Identity = identity

	// Write config file to repo
	if err = repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("saving new key to config (%v)", err)
	}
	return nil
}

func shorten(s string) string {
	if len(s) > 10 {
		s = s[:10]
	}
	return s
}
