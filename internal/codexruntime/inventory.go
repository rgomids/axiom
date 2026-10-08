package codexruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
)

// SkillSetState is the read-only compatibility fact for Axiom-owned skills.
// Skills that Axiom does not own in the shared root are never inspected.
type SkillSetState string

const (
	SkillSetAbsent      SkillSetState = "absent"
	SkillSetCurrent     SkillSetState = "current"
	SkillSetUpgradable  SkillSetState = "upgradable"
	SkillSetForeign     SkillSetState = "foreign"
	SkillSetInterrupted SkillSetState = "interrupted"
)

type InventorySkill struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256,omitempty"`
	Bytes  int64  `json:"bytes,omitempty"`
	State  string `json:"state"`
}

type Inventory struct {
	State   SkillSetState    `json:"state"`
	Receipt string           `json:"receipt"`
	Skills  []InventorySkill `json:"skills"`
}

// Inventory classifies the Axiom skill names and receipt without creating
// the root, taking the install lock, or modifying any skill.
func (s Service) Inventory(ctx context.Context) (Inventory, error) {
	return s.inventory(ctx, false)
}

// OwnershipInventory includes retired names solely for compatibility inspection
// and preservation. It is not the installed Runtime discovery surface.
func (s Service) OwnershipInventory(ctx context.Context) (Inventory, error) {
	return s.inventory(ctx, true)
}

func (s Service) inventory(ctx context.Context, historical bool) (Inventory, error) {
	names := skillNames
	if historical {
		names = ownershipSkillNames()
	}
	if err := ctx.Err(); err != nil {
		return Inventory{}, err
	}
	result := Inventory{State: SkillSetAbsent, Receipt: "absent", Skills: make([]InventorySkill, 0, len(names))}
	if _, err := os.Lstat(s.root); os.IsNotExist(err) {
		for _, name := range names {
			result.Skills = append(result.Skills, InventorySkill{Name: name, State: "missing"})
		}
		return result, nil
	} else if err != nil || !skillRootDirectory(s.root) {
		result.State, result.Receipt = SkillSetForeign, "unavailable"
		return result, nil
	}
	foreign, present, current := false, 0, 0
	for _, name := range names {
		skill := InventorySkill{Name: name, State: "missing"}
		if _, err := os.Lstat(filepath.Join(s.root, name)); err == nil {
			present++
			content, ok := singleSkillContent(filepath.Join(s.root, name))
			expected, readErr := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
			switch {
			case !ok:
				skill.State, foreign = "foreign", true
			case readErr == nil && string(content) == string(expected):
				skill.State = "current"
				current++
			case s.integration.matchesLegacyInstalled(s.root, name):
				skill.State = "legacy"
			default:
				skill.State, foreign = "foreign", true
			}
			if ok {
				digest := sha256.Sum256(content)
				skill.SHA256, skill.Bytes = hex.EncodeToString(digest[:]), int64(len(content))
			}
		} else if !os.IsNotExist(err) {
			skill.State, foreign = "foreign", true
		}
		result.Skills = append(result.Skills, skill)
	}
	receiptPath := filepath.Join(s.root, receiptName)
	if _, err := os.Lstat(receiptPath); err == nil {
		receipt, receiptErr := s.integration.receipt(s.root)
		switch {
		case receiptErr == nil && matchesPrivateFile(receiptPath, receipt):
			result.Receipt = "current"
		case s.integration.matchesLegacyReceipt(s.root):
			result.Receipt = "legacy"
		default:
			result.Receipt, foreign = "foreign", true
		}
	} else if !os.IsNotExist(err) {
		result.Receipt, foreign = "foreign", true
	}
	for _, name := range retiredSkillNames {
		if historical {
			break
		}
		if _, err := os.Lstat(filepath.Join(s.root, name)); os.IsNotExist(err) {
			continue
		}
		present++
		if !s.integration.matchesLegacyInstalled(s.root, name) {
			foreign = true
		}
	}
	if _, err := os.Lstat(filepath.Join(s.root, ".axiom-skill-set-receipt-stage")); err == nil {
		result.State = SkillSetInterrupted
		return result, nil
	}
	switch {
	case foreign:
		result.State = SkillSetForeign
	case present == 0 && result.Receipt == "absent":
		result.State = SkillSetAbsent
	case current == len(skillNames) && present == len(skillNames) && result.Receipt == "current":
		result.State = SkillSetCurrent
	default:
		result.State = SkillSetUpgradable
	}
	return result, nil
}
