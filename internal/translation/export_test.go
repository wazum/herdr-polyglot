package translation

// Remembered reports how many sentences the store holds, so a test can show it
// stays bounded.
func Remembered(translator Translator) int {
	store, ok := translator.(*segmented)
	if !ok {
		return 0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.known)
}
