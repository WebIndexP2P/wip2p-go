package account

import (
	"log"
	"encoding/hex"

	bolt "go.etcd.io/bbolt"
	"github.com/ethereum/go-ethereum/common"

	"code.wip2p.com/mwadmin/wip2p-go/core/globals"
	"code.wip2p.com/mwadmin/wip2p-go/invite"
)


// this function is called upon recieving a new bundle with valid invites in it
// we update the accounts "Invited" and either add or remove "Inviters" for each invited account
func (a *AccountStruct) UpdateInvited(invites []invite.Invite, tx *bolt.Tx) []AccountStruct {

	if globals.DebugLogging {
		log.Printf("account->UpdateInvited() for %s\n", a.Address.String())
	}

	updatedAccounts := make([]AccountStruct, 0)

	// take a copy of the original invite list
	originalInvitedIndexed := make(map[[20]byte]invite.Invite)
	for _, tmpInvite := range a.Invited {
		originalInvitedIndexed[tmpInvite.Account] = tmpInvite
	}

	// update the invite list to the new list provided
	a.Invited = invites

	// loop through and determine which invites are "active"
	for idx, tmpInvite := range a.Invited {

		if tmpInvite.Timestamp <= a.ActiveInviter.AssignedTimestamp() {
			log.Printf("Invite timestamp before inviters own timestamp\n")
			a.Invited[idx].Active = false
			continue
		}

		// we got here, so invite must be active
		a.Invited[idx].Active = true

		// check if the invite was updated in a way that requires a recursiveUpdateStatus call
		inviteMsg := ""
		_, found := originalInvitedIndexed[tmpInvite.Account]
		if !found {
			inviteMsg = "new invite for %+v\n"
		}
		if tmpInvite.Equals(originalInvitedIndexed[tmpInvite.Account]) == false {

			if inviteMsg == "" {
				inviteMsg = "invite has changed for %v\n"
			}
			log.Printf(inviteMsg, tmpInvite.Account.String())

			invitedAccountFromDb, found := FetchAccountFromDb(tmpInvite.Account, tx, true)
			if !found {
				log.Printf("Account %+v created\n", tmpInvite.Account)
				invitedAccountFromDb = AccountStruct{Address: tmpInvite.Account}
			}

			if (a.ActiveTimestamp() == 0 && a.ActiveLevel() == 0) {
				log.Fatal("account missing valid inviter details")
			}

			inviter := invite.Inviter{
				InviterAccount: a.Address,
				InvitersActiveTimestamp: a.ActiveTimestamp(),
				InvitersActiveLevel: a.ActiveLevel(),
				Timestamp: tmpInvite.Timestamp,
				LvlGap: tmpInvite.LvlGap,
			}
			invitedAccountFromDb.AddInviter(inviter)
			updatedAccounts = append(updatedAccounts, invitedAccountFromDb)
		}

		// invite is active, remove from dropped
		delete(originalInvitedIndexed, tmpInvite.Account)
	}

	// any accounts remaining in originalInvitedIndexed are considered dropped
	for key := range originalInvitedIndexed {

		if originalInvitedIndexed[key].Timestamp <= a.ActiveInviter.AssignedTimestamp() {
			//log.Printf("Existing invite was never active, skipping removal\n")
			continue
		}

		log.Printf("Dropped invite for 0x%v\n", hex.EncodeToString(key[:]))

		invitedAccountFromDb, success := FetchAccountFromDb(common.BytesToAddress(key[:]), tx, false)
		if !success {
			log.Fatal("Should not happen")
		}
		wasRemoved := invitedAccountFromDb.RemoveInviter(a.Address)
		if !wasRemoved {
			log.Fatal("could not remove invite")
		}

		updatedAccounts = append(updatedAccounts, invitedAccountFromDb)
	}

	return updatedAccounts
	// loop through dropped
}
