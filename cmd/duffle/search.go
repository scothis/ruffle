package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/renstrom/fuzzysearch/fuzzy"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/scothis/ruffle/pkg/bundle"
	"github.com/scothis/ruffle/pkg/repo/remote"
)

// BundleList is a list of bundle references.
// Implements a sorter on Name.
type BundleList []*bundle.Bundle

var validOutputs = []string{"table", "json"}

// Len returns the length.
func (bl BundleList) Len() int { return len(bl) }

// Swap swaps the position of two items in the versions slice.
func (bl BundleList) Swap(i, j int) { bl[i], bl[j] = bl[j], bl[i] }

// Less returns true if the version of entry a is less than the version of entry b.
func (bl BundleList) Less(a, b int) bool {
	return strings.Compare(bl[a].Name, bl[b].Name) < 1
}

func newSearchCmd(w io.Writer) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Hidden: true,
		Use:    "search",
		Short:  "perform a fuzzy search on available bundles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrUnderConstruction
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&output, "output", "o", "table", fmt.Sprintf("Specify an output format, one of: %v", validOutputs))

	return cmd
}

func search(keywords []string) (BundleList, error) {
	foundBundles := BundleList{}

	url := &url.URL{
		Scheme: "https",
		Host:   "hub.cnlabs.io",
		Path:   remote.IndexPath,
	}

	log.Debugf("Searching %s...", url.String())

	// if no keywords are given, display all available bundles
	if len(keywords) == 0 {
		return searchRepo(url)
	}
	for _, keyword := range keywords {
		resp, err := http.Get(url.String())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request to %s responded with a non-200 status code: %d", url.String(), resp.StatusCode)
		}

		index, err := remote.LoadIndexReader(resp.Body)
		if err != nil {
			return nil, err
		}

		var found = make(map[string]bool)
		names := make([]string, 0, len(index.Entries))
		for name := range index.Entries {
			names = append(names, name)
		}
		for _, foundName := range fuzzy.Find(keyword, names) {
			found[foundName] = true
		}
		// also check if the latest version of each bundle has a matching keyword
		for _, name := range names {
			for _, bundleKeyword := range index.Entries[name][0].Keywords {
				if bundleKeyword == keyword {
					found[name] = true
				}
			}
		}
		for n := range found {
			for name := range index.Entries {
				if n == name {
					foundBundles = append(foundBundles, index.Entries[name][0])
				}
			}
		}
	}
	return foundBundles, nil
}

func searchRepo(url *url.URL) (BundleList, error) {
	resp, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request to %s responded with a non-200 status code: %d", url.String(), resp.StatusCode)
	}

	index, err := remote.LoadIndexReader(resp.Body)
	if err != nil {
		return nil, err
	}

	bundles := make(BundleList, 0, len(index.Entries))
	for _, entry := range index.Entries {
		bundles = append(bundles, entry[0])
	}
	return bundles, nil
}
