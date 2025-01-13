//go:build integration
// +build integration

package releaseAnalyzer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/im7mortal/release-analyzer/pkg/releaseAnalyzer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFullProcess(t *testing.T) {
	ctx := context.Background()

	deltas, err := releaseAnalyzer.Process(ctx, "apache", "airflow", "v2.8.3", "v2.9.2")
	assert.NoError(t, err)
	assert.NotNil(t, deltas)

	b, _ := json.MarshalIndent(deltas, " ", "  ")
	fmt.Println(string(b))

	tagNotExist := "v2.8.7777777777"
	deltas, err = releaseAnalyzer.Process(ctx, "apache", "airflow", tagNotExist, "v2.9.2")
	assert.Error(t, err)
	assert.Nil(t, deltas)

	println(err.Error())

}
