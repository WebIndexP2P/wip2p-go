package account

import (
	"log"
	"bytes"
	"strings"
	"encoding/hex"

	bolt "go.etcd.io/bbolt"

	//"code.wip2p.com/mwadmin/wip2p-go/util"
	"code.wip2p.com/mwadmin/wip2p-go/names"
	"code.wip2p.com/mwadmin/wip2p-go/invite"
	"code.wip2p.com/mwadmin/wip2p-go/merklehead"
	"code.wip2p.com/mwadmin/wip2p-go/core/globals"
	"code.wip2p.com/mwadmin/wip2p-go/deltalog"
)

func (a *AccountStruct) RecurseUpdateStatus(tx *bolt.Tx) int { //returns didChange

	if globals.DebugLogging {
		log.Printf("RecurseUpdateStatus for %s\n", a.Address.String())
	}

	origEnabled := a.Enabled
	accountsUpdated := 0
	var newInviter invite.Inviter

	// only update active inviter if not root
	if bytes.Equal(a.Address[:], globals.RootAccount[:]) {

		log.Printf("root account, no need to UpdateActiveInviter\n")

		if len(a.Invited) > 0 {
			a.CheckAllInvited(tx)
		}

	} else {

		if len(a.Inviters) == 0 {
			accountsDisabledCount := a.DisableAccount(tx)
			a.SaveToDb(tx)
			return accountsDisabledCount
		} else if len(a.Inviters) == 1 {
			newInviter = a.Inviters[0]
		} else {
			// check the inviters
			newInviter = a.DetermineActiveInviter(tx)
		}
		a.Enabled = true

		inviterChanged := false

		if newInviter.Equals(a.ActiveInviter) == false {
			inviterChanged = true

			if origEnabled && a.ActiveInviter.AssignedLevel() != newInviter.AssignedLevel() && a.RootMultihash != nil {
				// update merklehead
				merklehead.Remove(strings.ToLower(a.Address.String()[2:]), tx)
				merklehead.Add(strings.ToLower(a.Address.String()[2:]), a.Timestamp, tx)
			}

			a.ActiveInviter = newInviter
			accountsUpdated++
		}

		// account was disabled and is now enabled.. loop through all invites
		if origEnabled == false && len(a.RootMultihash) > 0 {
			if a.Timestamp > newInviter.Timestamp {
				accountsUpdated += a.ReenableWithContent(tx)
			} else {
				a.RootMultihash = nil
				a.Timestamp = 0
				a.Signature = nil
			}
		}

		if origEnabled && newInviter.Timestamp > a.Timestamp && a.RootMultihash != nil {
			if globals.DebugLogging {
				log.Printf("remove now invalid bundle data due to timestamp conflict\n")
			}

			// remove the merklehead
			if a.RootMultihash != nil {
				merklehead.Remove(strings.ToLower(a.Address.String()[2:]), tx)
			}

			// remove the content
			a.RootMultihash = nil
			a.Timestamp = 0
			a.Signature = nil

			// update Sequence index
			deltalog.Update(a.Address, true, tx)
		}

		if inviterChanged {
			if len(a.Invited) > 0 {
				accountsUpdated += a.CheckAllInvited(tx)
			}
		}

	}

	a.SaveToDb(tx)
	return accountsUpdated
}

func (a *AccountStruct) CheckAllInvited(tx *bolt.Tx) int {

	if globals.DebugLogging {
		log.Printf("CheckAllInvited() %+v\n", a.Address.String())
	}

	// loop through all invited
	// check the lvl and timestamps against this accounts ActiveLevel and ActiveTimestamp
	//  if no longer valid, recurse into invited account

	for _, invited := range a.Invited {

		if globals.DebugLogging {
			log.Printf("CheckAllInvited() Loop->Invited -> %+v\n%+v\n", invited.Account.String(), invited)
		}


		if !invited.Active {
			if (a.ActiveInviter.Timestamp < invited.Timestamp) {

				invited.Active = true

				inviter := invite.Inviter{
					InviterAccount: a.Address,
					InvitersActiveTimestamp: a.ActiveTimestamp(),
					InvitersActiveLevel: a.ActiveLevel(),
					Timestamp: invited.Timestamp,
					LvlGap: invited.LvlGap,
				}

				childAccount, success := FetchAccountFromDb(invited.Account, tx, true)
				if !success {
					//log.Printf("Need to create %+v\n", childAccount)
					childAccount = AccountStruct{Address: invited.Account}
				}
				childAccount.AddInviter(inviter)
				childAccount.RecurseUpdateStatus(tx)
			}
		} else {
			childAccount, success := FetchAccountFromDb(invited.Account, tx, false)
			if !success {
				log.Printf("%+v\n", a.Address.String())
				log.Printf("CheckAllInvited() - error missing account %+v\n", invited.Account.String())
				panic("missing account")
			}
			childAccount.RecurseUpdateStatus(tx)
		}
	}

	return 0
}

func (a *AccountStruct) DisableAccount(tx *bolt.Tx) int {

	// when an account is disabled it will not be returned when requested
	// however its multihash data is still available until the disabled account is purged and there are no other refs

	//log.Printf("Disabling account %v\n", a.Address.String())

	if len(a.Inviters) > 0 {
		log.Fatal("Cant disable an account with existing inviters")
	}

	// obviously
	a.Enabled = false

	// remove from merklehead
	if len(a.RootMultihash) > 0 {
		address := hex.EncodeToString(a.Address.Bytes())
		merklehead.Remove(address, tx)

		// update delta log
		seqNo := deltalog.Update(a.Address, true, tx)
		globals.LatestSequenceNo = uint(seqNo)
	}

	a.ActiveInviter = invite.Inviter{}

	totalChanged := 1
	for _, invited := range a.Invited {
		childAccount, success := FetchAccountFromDb(invited.Account, tx, false)
		if !success {
			log.Fatal("DisableAccount() - missing account")
		}
		childAccount.RemoveInviter(a.Address)
		totalChanged += childAccount.RecurseUpdateStatus(tx)
	}

	names.RemoveAllForAddress(a.Address[:], tx)

	return totalChanged
}

func (a *AccountStruct) ReenableWithContent(tx *bolt.Tx) int {
	if globals.DebugLogging {
		log.Printf("Reenable account %v\n", a.Address.String())
	}

	// assume a.Inviters was empty and now contains one Inviters
	// assume a.Invited still accurate
	// loop through and determine which invites are "active"
	copyInvited := a.Invited
	a.Invited = make([]invite.Invite, 0)
	updatedAccounts := a.UpdateInvited(copyInvited, tx)
	totalAccountsUpdated := 1
	if len(updatedAccounts) > 0 {
		for _, updatedAccount := range updatedAccounts {
			accountsUpdated := updatedAccount.RecurseUpdateStatus(tx) // disable accounts, update active inviter, check bundle timestamp, recheck invites
			totalAccountsUpdated += accountsUpdated
		}
		log.Printf("%v accounts updated\n", totalAccountsUpdated)
	}

	log.Printf("FIXME: if one or more accounts updated, send merklehead update notice\n")

	// save the paste seq
	seqNo := deltalog.Update(a.Address, false, tx)
	globals.LatestSequenceNo = uint(seqNo)

	// add to merklehead
	addressTail := hex.EncodeToString(a.Address.Bytes())
	err := merklehead.Add(addressTail, a.Timestamp, tx)
	if err != nil {
		log.Fatal(err)
	}

	return totalAccountsUpdated
}
