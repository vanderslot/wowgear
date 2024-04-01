package wowgear

import (
	"fmt"
	"log/slog"
)

type Slot struct {
	Type        string
	DisplayName string
}

type Equipment struct {
	Slot *Slot
	Item *Item
}

type Build struct {
	Equipments []Equipment
	StatList   *StatList
	SetBonuses []*SetBonus
}

type Set struct {
	Id          string     `json:"id,omitempty"`
	DisplayName string     `json:"displayName,omitempty"`
	Bonuses     []SetBonus `json:"bonuses,omitempty"`
}

type SetBonus struct {
	Amount int      `json:"amount,omitempty"`
	Bonus  Property `json:"bonus,omitempty"`
	Value  float64  `json:"value,omitempty"`
}

var HighestValueFound float64

var BestBuildFound *Build

func InitBuild() *Build {
	return &Build{
		Equipments: []Equipment{
			{
				Slot: &Slot{
					Type:        "head",
					DisplayName: "Head",
				},
			},
			{
				Slot: &Slot{
					Type:        "neck",
					DisplayName: "Neck",
				},
			},
			{
				Slot: &Slot{
					Type:        "shoulder",
					DisplayName: "Shoulder",
				},
			},
			{
				Slot: &Slot{
					Type:        "back",
					DisplayName: "Back",
				},
			},
			{
				Slot: &Slot{
					Type:        "chest",
					DisplayName: "Chest",
				},
			},
			{
				Slot: &Slot{
					Type:        "wrist",
					DisplayName: "Wrist",
				},
			},
			{
				Slot: &Slot{
					Type:        "weapon",
					DisplayName: "Main Hand Weapon",
				},
			},
			{
				Slot: &Slot{
					Type:        "weapon",
					DisplayName: "Off Hand Weapon",
				},
			},
			{
				Slot: &Slot{
					Type:        "ranged",
					DisplayName: "Ranged Weapon",
				},
			},
			{
				Slot: &Slot{
					Type:        "hands",
					DisplayName: "Hands",
				},
			},
			{
				Slot: &Slot{
					Type:        "waist",
					DisplayName: "Waist",
				},
			},
			{
				Slot: &Slot{
					Type:        "legs",
					DisplayName: "Legs",
				},
			},
			{
				Slot: &Slot{
					Type:        "feet",
					DisplayName: "Feet",
				},
			},
			{
				Slot: &Slot{
					Type:        "finger",
					DisplayName: "Finger 1",
				},
			},
			{
				Slot: &Slot{
					Type:        "finger",
					DisplayName: "Finger 2",
				},
			},
			{
				Slot: &Slot{
					Type:        "trinket",
					DisplayName: "Trinket 1",
				},
			},
			{
				Slot: &Slot{
					Type:        "trinket",
					DisplayName: "Trinket 2",
				},
			},
		},
	}
}

var Combinations int
var Debug bool

func (b *Build) GetValue(sets []*Set) (float64, error) {
	total := 0.0
	Combinations++

	if Debug {
		fmt.Println(b.AsString())
	}

	for _, eq := range b.Equipments {
		itemValue, err := b.getItemValue(eq.Item)
		if err != nil {
			return 0, err
		}
		total += itemValue
	}
	b.SetBonuses = []*SetBonus{}
	for _, set := range sets {
		itemsInSet := b.countItemsInSet(set.Id)
		for _, setBonus := range set.Bonuses {
			if itemsInSet >= setBonus.Amount {
				value, err := getStatValue(setBonus.Bonus.StatCode, b.StatList)
				if err != nil {
					return 0, err
				}
				bonusValue := float64(setBonus.Bonus.Amount) * value
				setBonus.Value = bonusValue
				total += bonusValue
				b.SetBonuses = append(b.SetBonuses, &setBonus)
			}
		}
	}

	// Correct for hitcap
	hitValue := getHitValue(b.StatList)
	hit := b.getTotalHit()

	if hit > float64(b.StatList.HitCap) {
		total -= (hit - float64(b.StatList.HitCap)) * hitValue
	}

	return total, nil
}

func (b *Build) getTotalHit() float64 {
	result := 0.0
	for _, eq := range b.Equipments {
		if eq.Item != nil {
			for _, p := range eq.Item.Properties {
				if p.StatCode == "hit" {
					result += p.Amount
				}
			}
		}
	}
	return result
}

func getHitValue(stats *StatList) float64 {
	for _, stat := range stats.Stats {
		if stat.Code == "hit" {
			return stat.Value
		}
	}
	return 0.0
}

func (b *Build) AsString() string {
	result := ""
	for _, eq := range b.Equipments {
		if eq.Item != nil {
			if result != "" {
				result += ";"
			}
			result += eq.Item.DisplayName
		}
	}
	return result
}

func (b *Build) countItemsInSet(setId string) int {
	items := 0

	for _, eq := range b.Equipments {
		if eq.Item != nil && eq.Item.SetId == setId {
			items++
		}
	}
	return items
}

func (b *Build) getItemValue(item *Item) (float64, error) {
	if item == nil {
		return 0.0, nil
	}
	if item.Value > 0 {
		// Already calculated
		return item.Value, nil
	}
	for _, p := range item.Properties {
		val, err := getStatValue(p.StatCode, b.StatList)
		if err != nil {
			return 0, err
		}
		item.Value += float64(p.Amount) * val
	}
	return item.Value, nil
}

func (b *Build) Evaluate(slotNumber int, inv *Inventory) {
	var previous *Item
	if slotNumber > 0 {
		previous = b.Equipments[slotNumber - 1].Item
	}

	if slotNumber == len(b.Equipments) {
		value, err := b.GetValue(inv.Sets)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		if value > HighestValueFound {
			HighestValueFound = value
			BestBuildFound = &Build{
				Equipments: []Equipment{},
			}

			for _, eq := range b.Equipments {
				BestBuildFound.Equipments = append(BestBuildFound.Equipments, Equipment{
					Slot: eq.Slot,
					Item: eq.Item,
				})
			}
			BestBuildFound.SetBonuses = b.SetBonuses
		}
		return
	}

	slotType := b.Equipments[slotNumber].Slot.Type

	items := inv.getItemsForSlotType(slotType, previous)
	if len(items) == 0 {
		b.Evaluate(slotNumber+1, inv)
		return
	}

	for _, item := range items {
		b.Equipments[slotNumber].Item = item
		// Forward one slot
		next := slotNumber + 1

		if item.IsTwoHand {
			// Unequip off hand if main hand has 2H
			b.Equipments[next].Item = nil
			// And forward one extra slot
			next++
		}

		b.Evaluate(next, inv)
	}
}
