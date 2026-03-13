#[test_only]
module turret_low_hp_finisher::turret_tests;

use std::{bcs, unit_test::assert_eq};
use turret_low_hp_finisher::turret;
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
fun low_hp_finisher_prefers_damaged_targets() {
    let result = turret::build_priority_list_for_owner(
        OWNER_CHARACTER_ID,
        OWNER_TRIBE,
        candidate_list_bytes(vector[
            candidate(201, 920, 200, 20, 10, 5, false, 5, 1),
            candidate(202, 921, 200, 80, 80, 70, true, 10, 2),
            candidate(203, 922, OWNER_TRIBE, 10, 10, 10, false, 999, 2),
        ]),
    );
    assert_eq!(vector::length(&result), 2);
    assert_entry(vector::borrow(&result, 0), 201, 28005);
    assert_entry(vector::borrow(&result, 1), 202, 19010);
}