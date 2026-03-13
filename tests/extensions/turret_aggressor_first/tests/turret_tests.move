#[test_only]
module turret_aggressor_first::turret_tests;

use std::{bcs, unit_test::assert_eq};
use turret_aggressor_first::turret;
use world::turret::{Self, ReturnTargetPriorityList};

const OWNER_CHARACTER_ID: u32 = 108;
const OWNER_TRIBE: u32 = 100;

public struct TargetCandidateBcs has copy, drop {
    item_id: u64,
    type_id: u64,
    group_id: u64,
    character_id: u32,
    character_tribe: u32,
    hp_ratio: u64,
    shield_ratio: u64,
    armor_ratio: u64,
    is_aggressor: bool,
    priority_weight: u64,
    behaviour_change: u8,
}

fun candidate_list_bytes(entries: vector<TargetCandidateBcs>): vector<u8> { bcs::to_bytes(&entries) }

fun candidate(
    item_id: u64,
    character_id: u32,
    character_tribe: u32,
    hp_ratio: u64,
    shield_ratio: u64,
    armor_ratio: u64,
    is_aggressor: bool,
    priority_weight: u64,
    behaviour_change: u8,
): TargetCandidateBcs {
    TargetCandidateBcs {
        item_id,
        type_id: 1,
        group_id: 1,
        character_id,
        character_tribe,
        hp_ratio,
        shield_ratio,
        armor_ratio,
        is_aggressor,
        priority_weight,
        behaviour_change,
    }
}

fun assert_entry(entry: &ReturnTargetPriorityList, target_item_id: u64, priority_weight: u64) {
    assert_eq!(turret::return_target_item_id(entry), target_item_id);
    assert_eq!(turret::return_priority_weight(entry), priority_weight);
}

#[test]
fun aggressor_first_prioritizes_active_attackers() {
    let result = turret::build_priority_list_for_owner(
        OWNER_CHARACTER_ID,
        OWNER_TRIBE,
        candidate_list_bytes(vector[
            candidate(101, 900, 200, 85, 50, 40, true, 10, 2),
            candidate(102, 901, 200, 90, 90, 90, false, 25, 1),
            candidate(103, OWNER_CHARACTER_ID, 200, 90, 90, 90, true, 100, 2),
        ]),
    );
    assert_eq!(vector::length(&result), 2);
    assert_entry(vector::borrow(&result, 0), 101, 26685);
    assert_entry(vector::borrow(&result, 1), 102, 2375);
}