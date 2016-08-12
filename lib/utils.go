package dim

// ListContains checks a list of string contains a given string
func ListContains(list []string, search string) bool {
	for _, r := range list {
		if r == search {
			return true
		}
	}
	return false
}

// MapContainsNone checks a map of string does not contain any given string as key
/*func MapContainsNone(all map[string]string, forbidden []string) bool {
	if all == nil || len(all) == 0 || forbidden == nil || len(forbidden) == 0 {
		return true
	}
	for _, f := range forbidden {
		if all[f] != "" {
			return false
		}
	}
	return true
}

// SelectMapKeys retuns key of a map that exist in the keys list and that have non-empty values
func SelectMapKeys(all map[string]string, keys []string) []string {
	if all == nil || len(all) == 0 || keys == nil || len(keys) == 0 {
		return nil
	}

	selected := make([]string, len(keys))

	for _, k := range keys {
		if all[k] != "" {
			selected = append(selected, k)
		}
	}
	return selected

}*/

// MapMatchesAll checks first map contains all the second map elements with the same value
func MapMatchesAll(all, search map[string]string) bool {

	if all == nil || search == nil {
		return false
	}

	if len(all) >= len(search) {
		for k, v := range search {
			if all[k] != v {
				return false
			}
		}
		return true
	}
	return false

}

// Keys returns all the keys of the given map
func Keys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
