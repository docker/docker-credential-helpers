package keyctl

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestKeyctlHelper(t *testing.T) {
	helper := Keyctl{}

	// remove old stale values from previous failed run if any
	credsList, err := helper.List()
	if err != nil {
		t.Fatal(err)
	}

	for s, u := range credsList {
		if strings.Contains(s, "amazonecr") ||
			strings.Contains(s, "docker") {
			t.Logf("removing stale test entry for %s:%s", s, u)
			helper.Delete(s)
		}
	}

	creds := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io/v1:tag1",
		Username:  "nothing",
		Secret:    "mysecret",
	}
	helper.Add(creds)

	creds0 := &credentials.Credentials{
		ServerURL: "https://amazonecr.com/v1:tag2",
		Username:  "nothing0",
		Secret:    "mysecret0",
	}

	creds1 := &credentials.Credentials{
		ServerURL: "https://foobar.docker1.io/v1:tag3",
		Username:  "nothing1",
		Secret:    "mysecret1",
	}
	helper.Add(creds)
	helper.Add(creds0)
	helper.Add(creds1)

	credsList, err = helper.List()
	if err != nil {
		t.Fatal(err)
	}

	for s, u := range credsList {
		if !strings.Contains(s, "amazonecr") &&
			!strings.Contains(s, "docker") {
			t.Fatalf("unrecognized server name found Server: %s Username: %s ", s, u)
		}
		err = helper.Delete(s)
		if err != nil {
			t.Error(fmt.Errorf("error in deleting %s: %w", s, err))
		}
	}

	/* Read the list of credentials again */
	credsList, err = helper.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(credsList) != 0 {
		t.Fatalf("didn't delete all creds? %d", len(credsList))
	}
}
