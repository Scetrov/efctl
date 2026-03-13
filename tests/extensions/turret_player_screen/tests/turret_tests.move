#[test_only]
module turret_player_screen::turret_tests;

use std::{bcs, unit_test::assert_eq};
use turret_player_screen::turret;
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
fun player_screen_ignores_npcs_and_same_tribe_non_aggressors() {
    let result = turret::build_priority_list_for_owner(
        OWNER_CHARACTER_ID,
        OWNER_TRIBE,
        candidate_list_bytes(vector[
            candidate(301, 0, 0, 50, 50, 50, true, 30, 2),
            candidate(302, 930, OWNER_TRIBE, 60, 60, 60, false, 40, 1),
            candidate(303, 931, 200, 60, 60, 60, true, 50, 2),
            candidate(304, 932, 200, 60, 60, 60, false, 70, 1),
        ]),
    );
    assert_eq!(vector::length(&result), 2);
    assert_entry(vector::borrow(&result, 0), 303, 22050);
    assert_entry(vector::borrow(&result, 1), 304, 4070);
}