package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"Astramail/internal/service/graph"
)

const expectedInboxPath = "mailFolders/Inbox"

func TestResolveGraphFolderPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty folder defaults to Inbox", input: "", expected: expectedInboxPath},
		{name: "INBOX uppercase", input: "INBOX", expected: expectedInboxPath},
		{name: "inbox lowercase", input: "inbox", expected: expectedInboxPath},
		{name: "junk folder", input: "Junk", expected: "mailFolders/junkemail"},
		{name: "junk email with space", input: "junk email", expected: "mailFolders/junkemail"},
		{name: "drafts folder", input: "Drafts", expected: "mailFolders/drafts"},
		{name: "sent items folder", input: "Sent Items", expected: "mailFolders/sentitems"},
		{name: "trash folder", input: "Trash", expected: "mailFolders/deleteditems"},
		{name: "custom folder id", input: "custom-id-123", expected: "mailFolders/custom-id-123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := graph.ResolveGraphFolderPath(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestBuildGraphListURL(t *testing.T) {
	t.Parallel()

	t.Run("default list url has no raw spaces and contains orderby", func(t *testing.T) {
		t.Parallel()

		url := graph.BuildGraphListURL("INBOX", 20, 0, nil)

		assert.NotContains(t, url, " ", "URL must not contain raw whitespace")
		assert.Contains(t, url, expectedInboxPath+"/messages")
		assert.Contains(t, url, "$top=20")
		assert.Contains(t, url, "$skip=0")
		assert.Contains(t, url, "$count=true")
		assert.Contains(t, url, "$orderby=receivedDateTime%20desc")
		assert.Contains(t, url, "$select=id,subject,from,toRecipients,receivedDateTime,hasAttachments,bodyPreview")
	})

	t.Run("search query omits orderby per Graph specification", func(t *testing.T) {
		t.Parallel()

		q := "verify code"
		url := graph.BuildGraphListURL("INBOX", 10, 20, &q)

		assert.NotContains(t, url, " ", "URL must not contain raw whitespace")
		assert.Contains(t, url, "$search=")
		assert.NotContains(t, url, "$orderby", "Graph API does not allow $orderby when $search is used")
	})

	t.Run("junk folder url uses well-known name", func(t *testing.T) {
		t.Parallel()

		url := graph.BuildGraphListURL("Junk", 15, 0, nil)

		assert.NotContains(t, url, " ")
		assert.Contains(t, url, "mailFolders/junkemail/messages")
	})
}
