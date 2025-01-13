package releaseAnalyzer

import (
	"context"
	"fmt"
	"github.com/google/go-github/v53/github"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Delta represents the comparison between two releases.
type Delta struct {
	PreviousTag string  `json:"previous_tag"` // Tag of the previous release
	Tag         string  `json:"tag"`          // Current release tag
	Delta       float64 `json:"delta"`        // Change factor between current and previous release sizes

	url         *string // URL of the asset to compare
	size        int64   // Size of the asset in bytes
	err         error   // Error encountered while fetching the size
	discardFlag bool    // Indicates if this delta should be excluded from the final array
}

// TagNotFound is returned if the requested tag is not found in the release list.
type TagNotFound struct {
	err string
}

func (e *TagNotFound) Error() string {
	return e.err
}

// newTagNotFoundError creates a TagNotFound error with appropriate details.
func newTagNotFoundError(tagStartFound, tagEndFound bool, tagStart, tagEnd string) error {
	if !tagStartFound && !tagEndFound {
		return &TagNotFound{err: fmt.Sprintf("start tag %s and end tag %s not found", tagStart, tagEnd)}
	}
	if !tagStartFound {
		return &TagNotFound{err: fmt.Sprintf("start tag %s not found", tagStart)}
	}
	if !tagEndFound {
		return &TagNotFound{err: fmt.Sprintf("end tag %s not found", tagEnd)}
	}
	return nil // This case will not be used
}

// fetchReleases retrieves all releases for a repository, ensuring no releases are missed (e.g., older releases of previous versions).
func fetchReleases(ctx context.Context, owner, repo string, cl *http.Client) ([]*github.RepositoryRelease, error) {
	client := github.NewClient(cl)
	var allReleases []*github.RepositoryRelease

	options := &github.ListOptions{PerPage: 100}

	// Fetch all releases, handling pagination.
	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, options)
		if err != nil {
			println(err.Error())
			return nil, err
		}
		allReleases = append(allReleases, releases...)

		// Stop if there are no more pages.
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}

	return allReleases, nil
}

// transformRelease creates a Delta object from a GitHub release if the release contains an asset matching the required pattern.
func transformRelease(release *github.RepositoryRelease, previousTag string, discardFlag bool) *Delta {
	for _, asset := range release.Assets {
		if strings.HasSuffix(*asset.BrowserDownloadURL, ".tar.gz") &&
			!strings.Contains(*asset.BrowserDownloadURL, "source") {
			return &Delta{Tag: *release.TagName, PreviousTag: previousTag, url: asset.BrowserDownloadURL, discardFlag: discardFlag}
		}
	}
	return nil
}

// filterReleases filters releases by specified tags and converts them into Delta objects.
func filterReleases(allReleases []*github.RepositoryRelease, a, b string) ([]*Delta, error) {
	// Sort releases based on tag names after removing the "v" prefix.
	sort.Slice(allReleases, func(i, j int) bool {
		return strings.TrimPrefix(allReleases[i].GetTagName(), "v") < strings.TrimPrefix(allReleases[j].GetTagName(), "v")
	})

	tagStart := strings.TrimPrefix(a, "v")
	tagEnd := strings.TrimPrefix(b, "v")

	var tagStartFound, tagEndFound bool
	var deltas []*Delta

	for i, release := range allReleases {
		if release.TagName != nil {
			tag := strings.TrimPrefix(*release.TagName, "v")

			if tag == tagStart {
				tagStartFound = true
			}
			if tag == tagEnd {
				tagEndFound = true
			}

			if tag >= tagStart && tag <= tagEnd {
				var previousTag string
				if i != 0 {
					// Add a Delta for the previous tag to calculate delta values.
					if len(deltas) == 0 {
						deltas = append(deltas, transformRelease(allReleases[i-1], "", true))
					}
					previousTag = *allReleases[i-1].TagName
				}
				deltas = append(deltas, transformRelease(allReleases[i], previousTag, false))
			}
		}
	}

	if !tagStartFound || !tagEndFound {
		return nil, newTagNotFoundError(tagStartFound, tagEndFound, tagStart, tagEnd)
	}

	return deltas, nil
}

const headSizeTimeout = 10 * time.Second // Timeout for HEAD requests

// fetchFileSize performs a HEAD request to get the asset's size from the Content-Length header and updates the Delta.
func fetchFileSize(ctx context.Context, delta *Delta, wg *sync.WaitGroup) {
	if delta == nil { // Ensure Delta is not nil
		return
	}

	// Safeguard against panic in the goroutine.
	defer func() {
		if r := recover(); r != nil {
			delta.err = r.(error)
		}
		wg.Done() // Signal goroutine completion
	}()

	// Create a custom client with a timeout.
	httpClient := &http.Client{
		Timeout: headSizeTimeout,
	}

	// Create a HEAD request.
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, *delta.url, nil)
	if err != nil {
		delta.err = err
		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		delta.err = err
		fmt.Printf("Error fetching size for %s: %v\n", *delta.url, err)
		return
	}
	defer resp.Body.Close()

	// Check for non-200 status codes.
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Non-200 status for %s: %d\n", *delta.url, resp.StatusCode)
		return
	}

	// Parse Content-Length header.
	sizeHeader := resp.Header.Get("Content-Length")
	if sizeHeader == "" {
		fmt.Printf("Content-Length not found for %s\n", *delta.url)
		return
	}

	var size int64
	fmt.Sscanf(sizeHeader, "%d", &size) // Parse size from header.
	delta.size = size
}

// updateDeltas fetches the size for all Delta objects concurrently.
func updateDeltas(ctx context.Context, deltas []*Delta) {
	var wg sync.WaitGroup

	for _, delta := range deltas {
		wg.Add(1)
		go fetchFileSize(ctx, delta, &wg) // Fetch sizes concurrently
	}

	wg.Wait() // Wait for all goroutines to finish
}

// Process is the main function that orchestrates the fetching and processing of release data.
func Process(ctx context.Context, org, repo, tagStart, tagEnd string) ([]*Delta, error) {
	releases, err := fetchReleases(ctx, org, repo, &http.Client{
		Timeout: headSizeTimeout,
	})
	if err != nil {
		return nil, err
	}

	d, err := filterReleases(releases, tagStart, tagEnd)
	if err != nil {
		return nil, err
	}

	updateDeltas(ctx, d)

	for _, d := range d {
		if d.err != nil {
			return nil, err
		}
	}

	// Sort deltas by tag name just in case.
	sort.Slice(d, func(i, j int) bool { return strings.TrimPrefix(d[i].Tag, "v") < strings.TrimPrefix(d[j].Tag, "v") })

	// Calculate delta ratios for the releases.
	for i := range d {
		if i == 0 {
			d[i].Delta = 1 // First delta has no previous release to compare.
			continue
		}
		d[i].Delta = float64(d[i].size) / float64(d[i-1].size)
	}

	// Discard the first element if it was only used for delta calculations.
	if len(d) > 0 && d[0].discardFlag {
		d = d[1:]
	}

	return d, nil
}
