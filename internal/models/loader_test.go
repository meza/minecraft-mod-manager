package models

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllLoaders(t *testing.T) {
	expected := []Loader{
		BUKKIT, BUNGEECORD, CAULDRON, DATAPACK, FABRIC, FOLIA, FORGE, LITELOADER,
		MODLOADER, NEOFORGE, PAPER, PURPUR, QUILT, RIFT, SPIGOT, SPONGE, VELOCITY, WATERFALL,
	}
	actual := AllLoaders()
	assert.Equal(t, expected, actual, "AllLoaders should return all defined loaders")
}

func TestLoaderMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		loader   Loader
		expected string
	}{
		{"BUKKIT", BUKKIT, "bukkit"},
		{"BUNGEECORD", BUNGEECORD, "bungeecord"},
		{"CAULDRON", CAULDRON, "cauldron"},
		{"DATAPACK", DATAPACK, "datapack"},
		{"FABRIC", FABRIC, "fabric"},
		{"FOLIA", FOLIA, "folia"},
		{"FORGE", FORGE, "forge"},
		{"LITELOADER", LITELOADER, "liteloader"},
		{"MODLOADER", MODLOADER, "modloader"},
		{"NEOFORGE", NEOFORGE, "neoforge"},
		{"PAPER", PAPER, "paper"},
		{"PURPUR", PURPUR, "purpur"},
		{"QUILT", QUILT, "quilt"},
		{"RIFT", RIFT, "rift"},
		{"SPIGOT", SPIGOT, "spigot"},
		{"SPONGE", SPONGE, "sponge"},
		{"VELOCITY", VELOCITY, "velocity"},
		{"WATERFALL", WATERFALL, "waterfall"},
		{"Invalid", Loader("invalid"), "invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := json.Marshal(test.loader)
			assert.NoError(t, err, "unexpected error during marshaling")
			assert.Equal(t, fmt.Sprintf(`"%s"`, test.expected), string(actual), "marshaled value mismatch")
		})
	}
}

func TestLoaderUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Loader
	}{
		{"BUKKIT", `"bukkit"`, BUKKIT},
		{"BUNGEECORD", `"bungeecord"`, BUNGEECORD},
		{"CAULDRON", `"cauldron"`, CAULDRON},
		{"DATAPACK", `"datapack"`, DATAPACK},
		{"FABRIC", `"fabric"`, FABRIC},
		{"FOLIA", `"folia"`, FOLIA},
		{"FORGE", `"forge"`, FORGE},
		{"LITELOADER", `"liteloader"`, LITELOADER},
		{"MODLOADER", `"modloader"`, MODLOADER},
		{"NEOFORGE", `"neoforge"`, NEOFORGE},
		{"PAPER", `"paper"`, PAPER},
		{"PURPUR", `"purpur"`, PURPUR},
		{"QUILT", `"quilt"`, QUILT},
		{"RIFT", `"rift"`, RIFT},
		{"SPIGOT", `"spigot"`, SPIGOT},
		{"SPONGE", `"sponge"`, SPONGE},
		{"VELOCITY", `"velocity"`, VELOCITY},
		{"WATERFALL", `"waterfall"`, WATERFALL},
		{"Invalid", `"invalid"`, Loader("invalid")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var actual Loader
			err := json.Unmarshal([]byte(test.input), &actual)
			assert.NoError(t, err, "unexpected error during unmarshaling")
			assert.Equal(t, test.expected, actual, "unmarshaled value mismatch")
		})
	}
}
