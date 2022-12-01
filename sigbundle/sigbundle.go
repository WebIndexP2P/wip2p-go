package sigbundle

import (
  "fmt"
  "log"
  "time"
  "bytes"
  "errors"
  "strconv"
  "context"
  "encoding/hex"
  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-cid"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/common"

  carblockstore "github.com/ipld/go-car/v2/blockstore"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  //"code.wip2p.com/mwadmin/wip2p-go/util"
  "code.wip2p.com/mwadmin/wip2p-go/public"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/ipldwalk"
  "code.wip2p.com/mwadmin/wip2p-go/merklehead"
  "code.wip2p.com/mwadmin/wip2p-go/sigbundle/sigbundlestruct"
  accountlib "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/deltalog"
)

func ValidateAndSave(bundle sigbundlestruct.SigBundle, tx *bolt.Tx) (uint, []string, error) {

  accountsWithContentRemoved := make([]string, 0)
  var carRoot cid.Cid
  var carData *carblockstore.ReadOnly

  // check max doc size, currently limited to IPFS UnixFS single chunk size of 256KiB
  if len(bundle.Car) > 0 {
    buff := bytes.NewReader(bundle.Car)
    var err error
    carData, err = carblockstore.NewReadOnly(buff, nil)
    if err != nil {
      return 0, accountsWithContentRemoved, errors.New("error with car import")
    }
    roots, _ := carData.Roots()

    if len(roots) == 0 {
      return 0, accountsWithContentRemoved, errors.New("no root found")
    }
    if len(roots) > 1 {
      return 0, accountsWithContentRemoved, errors.New("too many roots found")
    }

    carRoot = roots[0]
  } else {
    for _, cborData := range bundle.CborData {
      if len(cborData) > 256 * 1024 {
    		return 0, accountsWithContentRemoved, errors.New("max size 256 KiB per file")
    	}
    }

    // dagCbor docs aim to stay under 64KiB because we want browsers to be able to load
    // and process this index data quickly and easily. This encourages user data to be
    // broken up into seperate docs by things like categories / time etc
    if len(bundle.CborData[0]) > 1024 * 64 {
      return 0, accountsWithContentRemoved, errors.New("max size 64KiB per cbor doc")
    }
  }

	// check timestamp not too far in the future
	cutOffTime := time.Now().UTC().Add(time.Minute * 5)
	cutOffTimeUnix := uint64(cutOffTime.Unix())

	if bundle.Timestamp > cutOffTimeUnix {
		return 0, accountsWithContentRemoved, errors.New("timestamp too far in the future")
	}

	// now check the signature is valid
	signedString := fmt.Sprintf("[%v,\"%v\"]", bundle.Timestamp, "0x" + hex.EncodeToString(bundle.RootMultihash))
	signedString = "\x19Ethereum Signed Message:\n" + strconv.Itoa(len(signedString)) + signedString
	signedBytes := []byte(signedString)
	hash := crypto.Keccak256Hash(signedBytes)

	tmpSig := make([]byte, len(bundle.Signature))
	copy(tmpSig, bundle.Signature)
	tmpSig[64] = tmpSig[64] - 27

	recPubKeyB, err := crypto.Ecrecover(hash.Bytes(), tmpSig)
	if err != nil {
		fmt.Println(err)
		return 0, accountsWithContentRemoved, errors.New("problem with signature")
	}

	recPubKey, _ := crypto.UnmarshalPubkey(recPubKeyB)
	//fmt.Printf("recPubKey %v\n", recPubKey)
	recAddress := crypto.PubkeyToAddress(*recPubKey)
	//fmt.Println(address)

	if bytes.Equal(recAddress.Bytes(), bundle.Account) == false {
		return 0, accountsWithContentRemoved, errors.New("signature invalid")
	}

	// Signature is all good, proceed

	// check cid for root_multihash matches
	pref := cid.Prefix{
		Version: 1,
		Codec: cid.DagCBOR,
		MhType: 0x12,
		MhLength: -1, // default length
	}

  var calc_cid cid.Cid
  var calc_mhash []byte
  if len(bundle.Car) > 0 {
    rootBlock, _ := carData.Get(nil, carRoot)
    calc_cid, _ = pref.Sum(rootBlock.RawData())
    calc_mhash = calc_cid.Hash()
  } else {
    calc_cid, _ = pref.Sum(bundle.CborData[0])
    calc_mhash = calc_cid.Hash()
  }

	if bytes.Equal(calc_mhash, bundle.RootMultihash) == false {
		return 0, accountsWithContentRemoved, errors.New("root_mulithash does not match data")
	}

	// Everything looks good, commit to db, broadcast and return a "ok" to sender
  log.Println("New bundle from " + recAddress.String())

  /*if len(bundle.Car) > 0 {
    fmt.Printf("%T %+v\n", carRoot, carRoot)
    fmt.Printf("%+v\n", calc_mhash)
    log.Fatal("boo")
  }*/


	// commit to db
  localTx := false
  if tx == nil {
    localTx = true
    tx = db.GetTx(true)
    defer db.RollbackTx(tx)
  }

  multihashChanged := true

	//accountB := ab.Get(bundle.Account)
	address := common.BytesToAddress(bundle.Account)
	account, success := accountlib.FetchAccountFromDb(address, tx, false)

	if !success {
		return 0, accountsWithContentRemoved, errors.New("account not found")
	}

	//fmt.Printf("found account %+v\n", account)

	if !account.Enabled {
		return 0, accountsWithContentRemoved, errors.New("account is not enabled")
	}

	// check timestamp is newer
	if bundle.Timestamp <= account.Timestamp {
		db.RollbackTx(tx)
		return 0, accountsWithContentRemoved, errors.New("timestamp must be more recent than previous")
	}

	// check timestamp is newer than invite date
	if bundle.Timestamp <= uint64(account.ActiveInviter.AssignedTimestamp()) {
		db.RollbackTx(tx)
		return 0, accountsWithContentRemoved, errors.New("timestamp must be more recent than invite")
	}

	// compare rootMultihashes
	if bytes.Equal(bundle.RootMultihash, account.RootMultihash) {
		//log.Println("FIXME: multihashes didnt change")
		multihashChanged = false
	}

  var updatedAccounts []accountlib.AccountStruct

  // assume the paste is committed from this point on
  //log.Println("New paste from " + recAddress.String())

  // save the doc/s to the db
  uploadSize := 0

  if multihashChanged {

    // calc multihash of every cborData file
    pref := cid.Prefix{
      Version: 1,
      Codec: cid.DagCBOR,
      MhType: 0x12,
      MhLength: -1, // default length
    }

    datasetMap := make(map[string][]byte)
    if len(bundle.Car) > 0 {

      ch, err := carData.AllKeysChan(context.Background())
      if err != nil {
        fmt.Printf("err %+v\n", err)
      }
      for tmpCid := range ch {
        //log.Fatal("boo2")
        //fmt.Printf("%+v\n", tmpCid)
        tmpBlock, err := carData.Get(nil, tmpCid)
        if err != nil {
          fmt.Printf("err %+v\n", err)
        }
        doubleCheckCid, _ := pref.Sum(tmpBlock.RawData())
        if !bytes.Equal(tmpCid.Hash(), doubleCheckCid.Hash()) {
          fmt.Printf("CID mismatch in CAR file for %s, using calculated CID %s instead\n", tmpCid, calc_cid)
        }
  	    datasetMap[ hex.EncodeToString(doubleCheckCid.Hash()) ] = tmpBlock.RawData()
      }
    } else {
      for a := 0; a < len(bundle.CborData); a++ {
    	  doubleCheckCid, _ := pref.Sum(bundle.CborData[a])
      	datasetMap[ hex.EncodeToString(doubleCheckCid.Hash()) ] = bundle.CborData[a]
      }
    }

    // will also enforce linked /wip2p/[ia] docs exist
    tmpCid := calc_cid
    if len(bundle.Car) > 0 {
      tmpCid = carRoot
    }

    docs, size, docsNotFound := ipldwalk.UpdateRoot(datasetMap, tmpCid, account.RootMultihash, tx)
    uploadSize = size
    log.Printf("Added %+v docs, size = %+v, missed %+v\n", docs, size, docsNotFound)

    //TODO: also need to cater for size quotas at some point
    if docsNotFound > 0 {
      account.SyncStatus = 2 // some missing
    } else {
      account.SyncStatus = 1 // complete
    }

		// if there was an original hash, remove the old data
		if account.RootMultihash != nil {
			// remove original data
			ipldwalk.RecursiveRemove(account.RootMultihash, tx)
		}

    // multihash changed so lets update it
    account.RootMultihash = bundle.RootMultihash

    // fetch the invites from the account docs
    newInvites := account.FetchInvitesFromDocs(tx)

    // apply new invites, keeping list of all changes
    // does not write changes to db
    updatedAccounts = account.UpdateInvited(newInvites, tx)

    account.Signature = bundle.Signature
    account.Timestamp = bundle.Timestamp

    // check for names
    namesAdded, dupNamesAdded := account.UpdateNames(tx)
    if namesAdded > 0 || dupNamesAdded > 0 {
      log.Printf("Names added: %v, Dup Names added: %v\n", namesAdded, dupNamesAdded)
    }


    // check if root and if private mode was changed
    if bytes.Equal(account.Address[:], globals.RootAccount[:]) {
	     public.UpdatePublicMode(bundle.CborData[0], tx)
    }

  }

  //nextSeq, _ := psb.NextSequence()
  nextSeq := deltalog.Update(address, false, tx)

  //fmt.Printf("next paste sequence is %+v\n", nextSeq)

  // calc byte size
  if uploadSize == 0 {
    if len(bundle.Car) > 0 {
      tmpBlock, _ := carData.Get(nil, carRoot)
      uploadSize = len(tmpBlock.RawData())
    } else {
      uploadSize = len(bundle.CborData[0])
    }
  }

	account.PasteSize = uint(uploadSize)
	account.PasteCount++

	//accountB, _ = json.Marshal(account)
	//ab.Put(bundle.Account, accountB)
	account.SaveToDb(tx)


  // add to merklehead
  addressTail := hex.EncodeToString(bundle.Account)
  err = merklehead.Add(addressTail, bundle.Timestamp, tx)
  if err != nil {
    db.RollbackTx(tx)
    return 0, accountsWithContentRemoved, err
  }

  // loop through all the invited/updated accounts and re-check their status
  //  this may add to the delta log and modify the merklehead
  if len(updatedAccounts) > 0 {
    //log.Printf("recursively checking %v child accounts for activeInviteTimestamp, activeLevel changes\n", len(updatedAccounts))
    totalAccountsUpdated := 0
    for _, updatedAccount := range updatedAccounts {
      accountsUpdated := updatedAccount.RecurseUpdateStatus(tx) // disable accounts, update active inviter, check bundle timestamp, recheck invites
      totalAccountsUpdated += accountsUpdated
    }
    log.Printf("%v accounts updated\n", totalAccountsUpdated)
  }


  if localTx {
    db.CommitTx(tx)
  }

  return uint(nextSeq), accountsWithContentRemoved, nil
}
