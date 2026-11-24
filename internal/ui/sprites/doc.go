// Package sprites renders the animated Pixel Yeast Cell mascot for the
// human-facing terminal UI. Sprite data is generated from the design-system
// source of truth (design_system/.../pixel.js via assets/sprites/pixel.json);
// see sprites_gen.go for the generated palette and frame grids.
//
//go:generate go run ../../../cmd/gen-sprites -in ../../../assets/sprites/pixel.json -out sprites_gen.go
package sprites
