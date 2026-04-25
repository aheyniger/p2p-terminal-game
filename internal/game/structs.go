package game

type PlayerId = string // probably a uuid

type Vec2 struct{ X, Y int } // just holds 2 values, likely for a position/coordinates
type Coord = Vec2            // alias for Vec2, specifically for coordinates. Maybe not needed?

type Direction byte

const (
	Left  Direction = 'L'
	Right Direction = 'R'
	Up    Direction = 'U'
	Down  Direction = 'D'
	None  Direction = 'N'
)

type Player struct {
	Id        PlayerId
	Addr      string // host and port (host should probably be IP address but maybe can be either IP address or hostname)
	Color     int32
	Pos       Vec2   // location of the player
	HeldBlock *Block // so we can separate blocks in the game world and identify each individually
}

type Block struct {
	ID  string // identifying a block, will be helpful for conflict resolution maybe
	Pos Vec2
}

type WorldState struct {
	Players map[PlayerId]*Player
	Blocks  map[Vec2]*Block
	Version uint64 // for lamport timestamps, keeping track of what game state is newest
}

// type MsgType string

// const (
// 	MsgPeerList    MsgType = "PEER_LIST"
// 	MsgStateUpdate MsgType = "STATE_UPDATE"
// 	// can add more as needed
// )
