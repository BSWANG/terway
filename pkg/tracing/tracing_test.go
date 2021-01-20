package tracing

import (
	"sort"
	"testing"
)

func TestPodMappings(t *testing.T) {
	mapping := PodMappings{
		{Name: "", Namespace: "", PodBindResID: "", RemoteResID: "dd", LocalResID: "dd", Valid: true}, // idle
		{Name: "default", Namespace: "a", PodBindResID: "aa", RemoteResID: "aa", LocalResID: "aa"},    // in use
		{Name: "", Namespace: "", PodBindResID: "", RemoteResID: "cc", LocalResID: "cc", Valid: true}, // idle
		{Name: "default", Namespace: "b", PodBindResID: "bb", RemoteResID: "bb", LocalResID: "bb"},    // in use
	}

	sort.Sort(mapping)

	camp := []string{"aa", "bb", "cc", "dd"}
	for i := range mapping {
		if mapping[i].RemoteResID != camp[i] {
			t.Errorf("mapping %#v", mapping[i])
		}
	}

}
