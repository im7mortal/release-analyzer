package releaseAnalyzer

import (
	"encoding/json"
	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestFilterReleases(t *testing.T) {

	var releases []*github.RepositoryRelease
	b, err := os.ReadFile("releases.json")
	assert.NoError(t, err)

	err = json.Unmarshal(b, &releases)
	assert.NoError(t, err)

	//for _, release := range releases {
	//	println(*release.TagName)
	//}

	deltas, err := filterReleases(releases, "v2.8.3", "v2.9.2")
	assert.NoError(t, err)
	println(len(deltas))

	b, err = json.Marshal(deltas)
	assert.NoError(t, err)
	println(string(b))

}
