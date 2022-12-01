package account

import (
	//"fmt"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	bolt "go.etcd.io/bbolt"

	"code.wip2p.com/mwadmin/wip2p-go/db"
	"code.wip2p.com/mwadmin/wip2p-go/invite"
	"code.wip2p.com/mwadmin/wip2p-go/ipldwalk"
	"code.wip2p.com/mwadmin/wip2p-go/merklehead"
	//"code.wip2p.com/mwadmin/wip2p-go/util"
	"code.wip2p.com/mwadmin/wip2p-go/core/globals"
	"code.wip2p.com/mwadmin/wip2p-go/sigbundle/sigbundlestruct"
	"code.wip2p.com/mwadmin/wip2p-go/deltalog"
	"code.wip2p.com/mwadmin/wip2p-go/names"
)

type AccountStruct struct {
	Address common.Address `json:"-"`

	Inviters           []invite.Inviter
	InviterAcceptOrder []common.Address
	Invited            []invite.Invite

	ActiveInviter invite.Inviter

	Signature     []byte
	Timestamp     uint64
	RootMultihash []byte
	DeltaSeqNo    uint64

	PasteSize  uint
	PasteCount uint

	SyncStatus uint //0 = not started, 1 = complete, 2 = some missing, 3 = size quota limit reached

	Enabled bool
}

func (a *AccountStruct) ActiveLevel() uint {
	return a.ActiveInviter.AssignedLevel()
}

func (a *AccountStruct) ActiveTimestamp() uint64 {
	return a.ActiveInviter.AssignedTimestamp()
}

// note when an account is disabled, or invites are changed, we need to run UpdateActiveInviter on all invited children
//  if no more inviters, account gets disabled

func (a *AccountStruct) RemoveContent(tx *bolt.Tx) {
	// remove all the docs
	ipldwalk.RecursiveRemove(a.RootMultihash, tx)

	// remove the delta log
	deltalog.Update(a.Address, true, tx)

	// remove merklehead
	address := hex.EncodeToString(a.Address.Bytes())
	merklehead.Remove(address, tx)

	// reset the content fields
	a.RootMultihash = nil
	a.Timestamp = 0
	a.Signature = nil
}


func (a *AccountStruct) AddInviter(inviter invite.Inviter) {

	for idx, v := range a.Inviters {
		if bytes.Equal(inviter.InviterAccount.Bytes(), v.InviterAccount.Bytes()) {
			a.Inviters[idx].Timestamp = inviter.Timestamp
			a.Inviters[idx].LvlGap = inviter.LvlGap
			return
		}
	}
	a.Inviters = append(a.Inviters, inviter)
}

func (a *AccountStruct) RemoveInviter(address common.Address) bool {

	var newInviters []invite.Inviter
	var bFound bool
	for _, tmpInviter := range a.Inviters {
		if bytes.Equal(address.Bytes(), tmpInviter.InviterAccount.Bytes()) == false {
			newInviters = append(newInviters, tmpInviter)
		} else {
			bFound = true
		}
	}
	a.Inviters = newInviters
	return bFound
}

func (a *AccountStruct) DetermineActiveInviter(tx *bolt.Tx) (invite.Inviter) {
	// determines what the new active inviter should be
	// compares ActiveLevel and Timestamp to what it was
	//  determines if all children need to be re-checked
	log.Printf("FIXME: check accept order\n")
	log.Printf("FIXME: fix invites same age\n")
	//log.Printf("DetermineActiveInviter() %+v\n", a.Address.String())

	oldestInviter := -1
	var bFoundMultiple bool
	for idx, inviter := range a.Inviters {

		if oldestInviter == -1 || inviter.AssignedTimestamp() < a.Inviters[oldestInviter].AssignedTimestamp() {
			oldestInviter = idx
			bFoundMultiple = false
		} else if inviter.AssignedTimestamp() == a.Inviters[oldestInviter].AssignedTimestamp() {
			bFoundMultiple = true
		}
	}

	if bFoundMultiple {
		log.Fatal("uh oh, two invites with same age")
	}

	return a.Inviters[oldestInviter]
}

func (a *AccountStruct) FetchInvitesFromDocs(tx *bolt.Tx) []invite.Invite {

	//log.Printf("FetchInvitesFromDocs - %+v\n", a.Address.String())

	invited := make([]invite.Invite, 0)

	invitesIface := ipldwalk.Get(a.RootMultihash, "/wip2p/i", tx)
	if invitesIface == nil {
		return invited
	}

	invitesArrayIface, success := invitesIface.([]interface{})
	if !success {
		return invited
	}

	invites := invite.Parse(invitesArrayIface, a.Address)
	log.Println("Found " + strconv.Itoa(len(invites)) + " invites")

	invited = make([]invite.Invite, len(invites))
	copy(invited, invites)

	return invited
}

func (a *AccountStruct) UpdateNames(tx *bolt.Tx) (namesAdded int, dupNamesAdded int) {

	finalNamesMap := make(map[string]common.Address)

	namesIface := ipldwalk.Get(a.RootMultihash, "/wip2p/n", tx)
	if namesIface != nil {
		namesMap := namesIface.(map[string]interface{})
		for k, v := range namesMap {
			accountBytes := v.([]byte)
			addr := common.BytesToAddress(accountBytes)
			finalNamesMap[k] = addr
		}
	}

	return names.Update(a.Address[:], a.ActiveInviter.Timestamp, finalNamesMap, tx)
}

func FetchAccountFromDb(address common.Address, tx *bolt.Tx, includeDisabled bool) (AccountStruct, bool) {

	//log.Printf("FetchAccountFromDb - %+v\n", address.String())

	if tx == nil {
		tx = db.GetTx(false)
		defer db.RollbackTx(tx)
	}

	ab := tx.Bucket([]byte("accounts"))
	accountB := ab.Get(address.Bytes())

	var account AccountStruct
	if accountB == nil {
		return account, false
	}

	json.Unmarshal(accountB, &account)
	account.Address = address

	if !account.Enabled && !includeDisabled {
		return AccountStruct{}, false
	}

	return account, true
}

func (a *AccountStruct) SaveToDb(tx *bolt.Tx) bool {
	if tx == nil {
		tx = db.GetTx(true)
		defer db.RollbackTx(tx)
	}

	isAccRoot := bytes.Equal(a.Address[:], globals.RootAccount[:])

	if !isAccRoot && a.Enabled && a.ActiveInviter.Timestamp == 0 {
		//panic("")
		log.Fatal("account requires active inviter to be set\n")
	}

	ab := tx.Bucket([]byte("accounts"))
	accountB, err := json.Marshal(a)
	if err != nil {
		log.Fatal(err)
	}
	ab.Put(a.Address.Bytes(), accountB)

	return true
}

func (a *AccountStruct) ExportSigBundle() (sigbundlestruct.SigBundle, error) {

	tx := db.GetTx(false)
	defer db.RollbackTx(tx)

	var bundle sigbundlestruct.SigBundle

	bundle.RootMultihash = a.RootMultihash
	bundle.Signature = a.Signature
	bundle.Timestamp = a.Timestamp

	doc := db.Doc{}
	docWithRefs, _ := doc.Get(a.RootMultihash, tx)

	if len(docWithRefs.Data) > 0 {
		bundle.CborData = make([][]byte, 0)
		bundle.CborData = append(bundle.CborData, docWithRefs.Data)
	}

	return bundle, nil
}
