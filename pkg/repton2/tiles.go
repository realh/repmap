package repton2

// All the tiles
const (
	T_BLANK = iota
	T_DIAMOND
	T_ROCK
	T_EGG
	T_SAFE
	T_KEY
	T_SPIRIT
	T_CAGE
	T_FLOWER
	T_BRICK_MID
	T_BRICK_TL
	T_BRICK_TR
	T_BRICK_L
	T_BRICK_R
	T_BRICK_T
	T_BRICK_B
	T_BRICK_BL
	T_BRICK_BR
	T_DIRT_1
	T_DIRT_2
	T_DIRT_3
	T_WALL_MID
	T_WALL_TL
	T_WALL_TR
	T_TRANSPORTER
	T_REPTON
	T_END
	T_SKULL
	T_WALL_BL
	T_WALL_BR
	T_PUZZLE
	T_SAVE
	T_BRICK_GROUND
	T_SKULL_RED

	N_TILES
)

// AnyColourTiles are those which are the same in all of Repton's colour themes
var AnyColourTiles = []int{
	T_BLANK,
	T_DIAMOND,
	T_ROCK,
	T_EGG,
	T_SAFE,
	T_CAGE,
	T_TRANSPORTER,
	T_REPTON,
	T_END,
	T_SKULL,
	T_SAVE,
	T_SKULL_RED,
}

// ColourThemedTiles are those which are different depending on colour theme
var ColourThemedTiles = []int{
	T_KEY,
	T_SPIRIT,
	T_FLOWER,
	T_BRICK_MID,
	T_BRICK_TL,
	T_BRICK_TR,
	T_BRICK_L,
	T_BRICK_R,
	T_BRICK_T,
	T_BRICK_B,
	T_BRICK_BL,
	T_BRICK_BR,
	T_DIRT_1,
	T_DIRT_2,
	T_DIRT_3,
	T_WALL_MID,
	T_WALL_TL,
	T_WALL_TR,
	T_WALL_BL,
	T_WALL_BR,
	T_SAVE,
	T_BRICK_GROUND,
}
