package commands

import "testing"

func TestIsOrgRepo(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"org/repo", true},
		{"my-org/my-repo", true},
		{"https://github.com/org/repo", false},
		{"git@github.com:org/repo.git", false},
		{"org/repo/extra", false},
		{"/absolute/path", false},
		{"noslash", false},
		{"", false},
	}
	for _, tc := range cases {
		got := isOrgRepo(tc.input)
		if got != tc.want {
			t.Errorf("isOrgRepo(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
