package dib

import (
	"github.com/go-directory/config"
)

/*
Config aliases [config.Config].
*/
type Config = config.Config

type dITConfig = config.DIT

/*
verifyRuntimeConfig returns an error following an attempt to read the
entries within "cn=dib,cn=config" to verify the needed runtime settings
for DB opening/creation are in-place and copacetic.
*/
func verifyRuntimeConfig(r *Config) (err error) {
	seen := make(map[string]struct{})
	expect := map[string]struct{}{
		"frontend": {},
		"state":    {},
		"schema":   {},
		"config":   {},
	}

	for i := 0; i < len(r.DIB.DITs); i++ {
		dit := r.DIB.DITs[i]

		var zeroDIT, isFE, ndd bool
		zeroDIT, isFE, ndd, err = verifyGlobalRuntimeConfig(&dit, seen)
		if err != nil {
			break
		}

		switch dit.Type {
		case 1: // Ephemeral
			err = verifyEphemeralRuntimeConfig(isFE, zeroDIT, ndd)

		case 2: // Persistent
			err = verifyPersistentRuntimeConfig(isFE, zeroDIT, ndd)

			r.DIB.DITs[i].DataFile = append(bTrimR(dit.DataDir, `/`), []byte(`/`)...)
			r.DIB.DITs[i].DataFile = append(dit.DataFile, dit.Name...)
			r.DIB.DITs[i].DataFile = append(dit.DataFile, []byte(`/data.db`)...)

			mkdir(string(dit.DataDir)+`/`+string(dit.Name), dit.Mode+0100)

		default:
			err = configError("Invalid DIT Type value; must be Ephemeral(1) or Persistent(2)")
		}

		// register DIT as "seen"
		seen[string(dit.Name)] = struct{}{}
	}

	// confirm that certain expected built-in contexts, such as
	// schema, frontend, et al., are indeed present in seen table.
	// If even one is missing, we will panic.
	for k := range expect {
		if _, ok := seen[k]; !ok {
			err = configError("Missing built-in DIT stub for '", k, "'")
			break
		}
	}

	return
}

func verifyGlobalRuntimeConfig(dit *dITConfig, seen map[string]struct{}) (zeroDIT, isFE, ndd bool, err error) {
	// ensure valid "friendly name"
	if dit.Name == nil || len(dit.Name) == 0 {
		err = configError("All DITs MUST have a unique alphanumeric name")
		return
	}

	// make sure we haven't seen this DIT config before
	if _, saw := seen[string(dit.Name)]; saw {
		err = configError("DIT ", string(dit.Name), " specified more than once")
		return
	}

	// ensure at least minimum privileges are satisfied
	if ownerCanRead := dit.Mode&0400 != 0; !ownerCanRead {
		err = configError(string(dit.Suffix), ": owner MUST have read privileges at a minimum")
		return
	} else if ownerCanWrite := dit.Mode&0600 != 0; !ownerCanWrite {
		dit.ReadOnly = true
	}

	zeroDIT = dit.Suffix == nil || len(dit.Suffix) == 0 // no suffix
	isFE = beq(dit.Name, []byte(`frontend`))            // name == frontend
	ndd = dit.DataDir == nil || len(dit.DataDir) == 0   // no data dir

	return
}

func verifyEphemeralRuntimeConfig(isFE, zeroDIT, ndd bool) (err error) {
	if zeroDIT && !isFE {
		err = configError("Only frontend (Root DSE) allows a zero suffix")
	} else if !zeroDIT && isFE {
		err = configError("Frontend requires a zero suffix")
	} else if !ndd {
		err = configError("Invalid Config.DataDir value for Ephemeral database (must be zero)")
	}

	return
}

func verifyPersistentRuntimeConfig(isFE, zeroDIT, ndd bool) (err error) {
	if isFE {
		err = configError("frontend (Root DSE) DIT Type MUST be Ephemeral")
	} else if zeroDIT {
		err = configError("Empty suffix can only serve frontend (Root DSE)")
	} else if ndd {
		err = configError("Invalid Config.DataDir value for Persistent database")
	}

	return
}
