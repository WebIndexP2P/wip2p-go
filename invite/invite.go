package invite

import (
	"bytes"
	"log"

	"github.com/ethereum/go-ethereum/common"

	"code.wip2p.com/mwadmin/wip2p-go/util"
	"code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

type Invite struct {
	Account   common.Address
	Timestamp uint64
	LvlGap    uint
	Active		bool
}

func (i *Invite) ConvertToMsg() messages.Invited {
	invMsg := messages.Invited{}
	invMsg.Account = i.Account.String()
	invMsg.Timestamp = uint(i.Timestamp)
	invMsg.LvlGap = uint(i.LvlGap)
	return invMsg
}


type Inviter struct {
	// store the inviters active timestamp and level for determining accept order
	InviterAccount common.Address
	InvitersActiveTimestamp uint64
	InvitersActiveLevel     uint

	Timestamp uint64
	LvlGap    uint
}

func (i *Inviter) AssignedLevel() uint {
	if util.AllZero(i.InviterAccount.Bytes()) {
		return 0
	} else {
		return i.InvitersActiveLevel + 1 + i.LvlGap
	}
}

func (i *Inviter) AssignedTimestamp() uint64 {
	return i.Timestamp
}

func (i *Inviter) Equals(c Inviter) bool {
	if bytes.Equal(i.InviterAccount[:], c.InviterAccount[:]) == false {
		return false
	}
	if i.Timestamp != c.Timestamp {
		return false
	}
	if i.LvlGap != c.LvlGap {
		return false
	}
	return true
}

func (i *Inviter) ConvertToMsg() messages.Inviter {
	invMsg := messages.Inviter{}
	invMsg.Account = i.InviterAccount.String()
	invMsg.Timestamp = uint(i.Timestamp)
	invMsg.LvlGap = uint(i.LvlGap)
	invMsg.InviterLevel = i.InvitersActiveLevel
	return invMsg
}

func Parse(invitesIface []interface{}, inviterAddress common.Address) []Invite {
	invites := make([]Invite, 0)
	uniqueInvites := make(map[common.Address]bool)

	for i := range invitesIface {

		tmpInvite := Invite{}

		inviteObj, ok := invitesIface[i].(map[string]interface{})
		if !ok {
			continue
		}

		varIface := inviteObj["account"]
		if varIface == nil {
			log.Println("invite missing account")
			continue
		}
		acctBytes, success := varIface.([]byte)
		if !success {
			log.Println("invited account appears invalid")
			continue
		}
		copy(tmpInvite.Account[:], acctBytes)

		if bytes.Equal(tmpInvite.Account[:], inviterAddress[:]) {
			log.Println("account cannot invite itself")
			continue
		}

		if _, found := uniqueInvites[tmpInvite.Account]; found {
			log.Println("account invited more than once, ignoring")
			continue
		}

		// add to the unique list
		uniqueInvites[tmpInvite.Account] = true

		varIface = inviteObj["timestamp"]
		if varIface == nil {
			log.Println("invite missing timestamp")
			continue
		}

		tmpVal, success := varIface.(int)
		if !success {
			log.Println("timestamp expects number")
			continue
		}
		tmpInvite.Timestamp = uint64(tmpVal)

		varIface = inviteObj["lvl"]
		if varIface != nil {
			tmpInvite.LvlGap = uint(varIface.(int))
		}

		varIface = inviteObj["lvlgap"]
		if varIface != nil {
			tmpInvite.LvlGap = uint(varIface.(int))
		}

		invites = append(invites, tmpInvite)
	}

	return invites
}

func (i *Invite) Equals(c Invite) bool {
	if bytes.Equal(i.Account[:], c.Account[:]) == false {
		return false
	}
	if i.Timestamp != c.Timestamp {
		return false
	}
	if i.LvlGap != c.LvlGap {
		return false
	}
	return true
}
