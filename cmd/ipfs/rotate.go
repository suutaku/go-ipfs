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
	ci "github.com/libp2p/go-libp2p-core/crypto"
	b58 "github.com/mr-tron/base58/base58"
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
		return err
	}
	defer repo.Close()

	// Read config file from repo
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	// Generate new identity
	identity, err := config.CreateIdentity(out, []options.KeyGenerateOption{
		options.Key.Size(nBitsForKeypair),
		options.Key.Type(algorithm),
	})
	if err != nil {
		return err
	}
	newPrivKey, err := identity.DecodePrivateKey("") //XXX
	if err != nil {
		return err
	}

	// Save old identity to keystore
	oldPrivKey, err := cfg.Identity.DecodePrivateKey("") //XXX
	if err != nil {
		return err
	}
	keystore := repo.Keystore()
	oldKey, err := pubKeyPrefix(oldPrivKey.GetPublic())
	if err != nil {
		return err
	}
	newKey, err := pubKeyPrefix(newPrivKey.GetPublic())
	if err != nil {
		return err
	}
	name := fmt.Sprintf("IdentityRotation-%s-from-%s-to-%s-at-%s",
		cfg.Identity.PeerID,
		oldKey,
		newKey,
		time.Now().Format(time.RFC822),
	)
	if err := keystore.Put(name, oldPrivKey); err != nil {
		return err
	}

	// Update identity
	cfg.Identity = identity

	// Write config file to repo
	return repo.SetConfig(cfg)
}

func pubKeyPrefix(k ci.PubKey) (string, error) {
	r, err := k.Raw()
	if err != nil {
		return "", err
	}
	return b58.Encode(r), nil
}
