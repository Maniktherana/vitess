/*
Copyright 2021 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"vitess.io/vitess/go/test/utils"
)

func Test_groupPRs(t *testing.T) {
	tests := []struct {
		name    string
		prInfos []pullRequestInformation
		want    map[string]map[string][]pullRequestInformation
	}{
		{
			name:    "Single PR info with no labels",
			prInfos: []pullRequestInformation{{Title: "pr 1", Number: 1}},
			want:    map[string]map[string][]pullRequestInformation{"Other": {"Other": []pullRequestInformation{{Title: "pr 1", Number: 1}}}},
		}, {
			name:    "Single PR info with type label",
			prInfos: []pullRequestInformation{{Title: "pr 1", Number: 1, Labels: []label{{Name: prefixType + "Bug"}}}},
			want:    map[string]map[string][]pullRequestInformation{"Bug fixes": {"Other": []pullRequestInformation{{Title: "pr 1", Number: 1, Labels: []label{{Name: prefixType + "Bug"}}}}}}},
		{
			name:    "Single PR info with type and component labels",
			prInfos: []pullRequestInformation{{Title: "pr 1", Number: 1, Labels: []label{{Name: prefixType + "Bug"}, {Name: prefixComponent + "VTGate"}}}},
			want:    map[string]map[string][]pullRequestInformation{"Bug fixes": {"VTGate": []pullRequestInformation{{Title: "pr 1", Number: 1, Labels: []label{{Name: prefixType + "Bug"}, {Name: prefixComponent + "VTGate"}}}}}}},
		{
			name: "Multiple PR infos with type and component labels", prInfos: []pullRequestInformation{
				{Title: "pr 1", Number: 1, Labels: []label{{Name: prefixType + "Bug"}, {Name: prefixComponent + "VTGate"}}},
				{Title: "pr 2", Number: 2, Labels: []label{{Name: prefixType + "Feature"}, {Name: prefixComponent + "VTTablet"}}}},
			want: map[string]map[string][]pullRequestInformation{"Bug fixes": {"VTGate": []pullRequestInformation{{Title: "pr 1", Number: 1, Labels: []label{{Name: prefixType + "Bug"}, {Name: prefixComponent + "VTGate"}}}}}, "Feature": {"VTTablet": []pullRequestInformation{{Title: "pr 2", Number: 2, Labels: []label{{Name: prefixType + "Feature"}, {Name: prefixComponent + "VTTablet"}}}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupPRs(tt.prInfos)
			utils.MustMatch(t, tt.want, got)
		})
	}
}

func TestLoadSummaryReadme(t *testing.T) {
	readmeFile, err := os.CreateTemp("", "*.md")
	require.NoError(t, err)

	readmeContent := `- New Gen4 feature
- Self hosted runners
- Bunch of features
`

	err = os.WriteFile(readmeFile.Name(), []byte(readmeContent), 0644)
	require.NoError(t, err)

	str, err := releaseSummary(readmeFile.Name())
	require.NoError(t, err)
	require.Equal(t, str, readmeContent)
}

func TestGenerateReleaseNotes(t *testing.T) {
	tcs := []struct {
		name                 string
		releaseNote          releaseNote
		expectedOut          string
		expectedOutChangeLog string
	}{
		{
			name:        "empty",
			releaseNote: releaseNote{},
			expectedOut: "# Release of Vitess \n",
		}, {
			name:        "with version number",
			releaseNote: releaseNote{Version: "v12.0.0"},
			expectedOut: "# Release of Vitess v12.0.0\n",
		}, {
			name:        "with announcement",
			releaseNote: releaseNote{Announcement: "This is the new release.\n\nNew features got added.", Version: "v12.0.0"},
			expectedOut: "# Release of Vitess v12.0.0\n" +
				"This is the new release.\n\nNew features got added.\n",
		}, {
			name:        "with announcement and known issues",
			releaseNote: releaseNote{Announcement: "This is the new release.\n\nNew features got added.", Version: "v12.0.0", KnownIssues: "* bug 1\n* bug 2\n"},
			expectedOut: "# Release of Vitess v12.0.0\n" +
				"This is the new release.\n\nNew features got added.\n" +
				"------------\n" +
				"## Known Issues\n" +
				"* bug 1\n" +
				"* bug 2\n\n",
		}, {
			name: "with announcement and change log",
			releaseNote: releaseNote{
				Announcement:      "This is the new release.\n\nNew features got added.",
				Version:           "v12.0.0",
				VersionUnderscore: "12_0_0",
				ChangeLog:         "* PR 1\n* PR 2\n",
				ChangeMetrics:     "optimization is the root of all evil",
				SubDirPath:        "changelog/12.0/12.0.0",
			},
			expectedOut: "# Release of Vitess v12.0.0\n" +
				"This is the new release.\n\nNew features got added.\n" +
				"------------\n" +
				"The entire changelog for this release can be found [here](https://github.com/vitessio/vitess/blob/main/changelog/12.0/12.0.0/changelog.md).\n" +
				"optimization is the root of all evil\n",
			expectedOutChangeLog: "# Changelog of Vitess v12.0.0\n" +
				"* PR 1\n" +
				"* PR 2\n\n",
		}, {
			name: "with only change log",
			releaseNote: releaseNote{
				Version:           "v12.0.0",
				VersionUnderscore: "12_0_0",
				ChangeLog:         "* PR 1\n* PR 2\n",
				ChangeMetrics:     "optimization is the root of all evil",
				SubDirPath:        "changelog/12.0/12.0.0",
			},
			expectedOut: "# Release of Vitess v12.0.0\n" +
				"The entire changelog for this release can be found [here](https://github.com/vitessio/vitess/blob/main/changelog/12.0/12.0.0/changelog.md).\n" +
				"optimization is the root of all evil\n",
			expectedOutChangeLog: "# Changelog of Vitess v12.0.0\n" +
				"* PR 1\n" +
				"* PR 2\n\n",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			outFileRn, err := os.CreateTemp("", "*.md")
			require.NoError(t, err)
			outFileChangelog, err := os.CreateTemp("", "*.md")
			require.NoError(t, err)
			err = tc.releaseNote.generate(outFileRn, outFileChangelog)
			require.NoError(t, err)
			all, err := os.ReadFile(outFileRn.Name())
			require.NoError(t, err)
			require.Equal(t, tc.expectedOut, string(all))
		})
	}
}

func TestGetStringForPullRequestInfos(t *testing.T) {
	testCases := []struct {
		name      string
		prPerType prsByType
		expected  string
	}{
		{
			name: "Single PR",
			prPerType: prsByType{
				"Feature": prsByComponent{
					"ComponentA": []pullRequestInformation{
						{Number: 1, Title: "PR 1", Labels: labels{{Name: "Type: Feature"}, {Name: "Component: ComponentA"}}},
					},
				},
			},
			expected: `### Feature
#### ComponentA
 * PR 1 [#1](https://github.com/vitessio/vitess/pull/1)
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getStringForPullRequestInfos(tc.prPerType)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestGetStringForKnownIssues(t *testing.T) {
	testCases := []struct {
		name     string
		issues   []knownIssue
		expected string
	}{
		{
			name: "Multiple Issues",
			issues: []knownIssue{
				{Number: 1, Title: "Issue 1"},
				{Number: 2, Title: "Issue 2"},
			},
			expected: ` * Issue 1 #1 
 * Issue 2 #2 
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getStringForKnownIssues(tc.issues)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestGroupAndStringifyPullRequest(t *testing.T) {
	testCases := []struct {
		name     string
		pris     []pullRequestInformation
		expected string
	}{
		{
			name: "Multiple PRs",
			pris: []pullRequestInformation{
				{Number: 1, Title: "PR 1", Labels: labels{{Name: "Type: Feature"}, {Name: "Component: ComponentA"}}},
				{Number: 2, Title: "PR 2", Labels: labels{{Name: "Type: Bug"}, {Name: "Component: ComponentB"}}},
			},
			expected: `### Bug fixes
#### ComponentB
 * PR 2 [#2](https://github.com/vitessio/vitess/pull/2)
### Feature
#### ComponentA
 * PR 1 [#1](https://github.com/vitessio/vitess/pull/1)
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := groupAndStringifyPullRequest(tc.pris)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestLoadKnownIssues(t *testing.T) {
	testCases := []struct {
		name           string
		release        string
		expectedIssues []knownIssue
		expectedErr    error
	}{
		{
			name:           "Valid Release",
			release:        "v1.2.3",
			expectedIssues: []knownIssue{{Number: 1, Title: "Issue 1"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := loadKnownIssues(tc.release)
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tc.expectedIssues), len(issues))

				// Sort both slices for comparison
				sort.Slice(issues, func(i, j int) bool {
					return issues[i].Number < issues[j].Number
				})
				sort.Slice(tc.expectedIssues, func(i, j int) bool {
					return tc.expectedIssues[i].Number < tc.expectedIssues[j].Number
				})

				for i := range issues {
					require.Equal(t, tc.expectedIssues[i], issues[i])
				}
			}
		})
	}
}
