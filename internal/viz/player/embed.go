// Package player is the Three.js interpreter of viz recipes against the clinic GLTF mouth.
// It does not author geometry. Nodes come from Teeth_Blender_Packed.blend → mouth.glb
// (TeethUpper, TeethLower, GumsUpper, GumsLower, jawLower, Saliva*, Tongue).
package player

import _ "embed"

//go:embed static/index.html
var HTML []byte

//go:embed static/player.js
var JS []byte

//go:embed static/cinematics.js
var Cinematics []byte
