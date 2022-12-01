package defaults

type details struct {
    Name string
    RootAccount string
    BootstrapPeers []string
}

type schedule struct {
  SizeIncrementDaysSchedule []uint
  LevelInviteLimitsSchedule []uint
}

func GetTrees() []details {
  return []details{details{
    Name: "General Sherman",
    RootAccount: "0x388d22ba6f190762b1dc5a813b845065b50c7da8",
    BootstrapPeers: []string{"wss://tulip.wip2p.com","ws://107.173.160.150"},
  }}
}

func GetSchedule() schedule {
  return schedule{
    SizeIncrementDaysSchedule: []uint{0, 1, 2, 4, 7, 12, 20, 32, 51, 82, 131, 209, 333, 530},
    LevelInviteLimitsSchedule: []uint{46, 35, 27, 21, 16, 12, 9, 7, 5, 0},
  }
}
