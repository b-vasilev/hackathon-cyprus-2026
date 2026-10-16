package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learningpack"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

func seedPracticeVersions(t *testing.T, portal *testPortal) {
	t.Helper()
	ctx := t.Context()
	if _, err := portal.database.ExecContext(ctx, "INSERT INTO llm_connections(id,name,created_at) VALUES('connection','Fixture','created')"); err != nil {
		t.Fatal(err)
	}
	if _, err := portal.database.ExecContext(ctx, "INSERT INTO llm_connection_revisions(connection_id,revision,encrypted_config,created_at) VALUES('connection',1,X'01','created')"); err != nil {
		t.Fatal(err)
	}
	for version := 1; version <= 2; version++ {
		revisionID := fmt.Sprintf("revision-%d", version)
		runID := fmt.Sprintf("run-%d", version)
		path := fmt.Sprintf("products/%s/versions/%d/passport.json", portal.productID, version)
		hash, err := portal.content.WriteJSON(path, passport.Passport{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = portal.database.ExecContext(ctx, `INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at)
 VALUES(?,?,'connection',1,'model','succeeded','complete','{}','created')`, runID, portal.productID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = portal.database.ExecContext(ctx, `INSERT INTO passport_revisions(id,product_id,run_id,version,schema_version,content_json,content_path,content_hash,created_at)
 VALUES(?,?,?,?,'2','{}',?,?,'created')`, revisionID, portal.productID, runID, version, path, hash)
		if err != nil {
			t.Fatal(err)
		}
		pack, err := json.Marshal(learningpack.Pack{Mission: learningpack.Mission{Title: fmt.Sprintf("Practice v%d", version)}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = portal.database.ExecContext(ctx, `INSERT INTO learning_packs(id,product_id,revision_id,status,content_json,published_content_json,created_at,updated_at,published_at)
 VALUES(?,?,?,'ready',?,?,'created','created',?)`, fmt.Sprintf("pack-%d", version), portal.productID, revisionID, string(pack), string(pack), fmt.Sprintf("2026-10-16T1%d:00:00Z", version))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestPublishedPracticeFollowsPublishedPassport(t *testing.T) {
	t.Parallel()
	portal := newTestPortal(t)
	seedPracticeVersions(t, portal)
	request := portal.request
	practice := "/api/products/" + portal.productID + "/learning-pack"
	passportVersion := func(version int, action string) string {
		return fmt.Sprintf("/api/passport-revisions/revision-%d/%s", version, action)
	}
	assertPractice := func(version int) {
		t.Helper()
		var pack learningpack.View
		body := request(http.MethodGet, practice, "", portal.learner, http.StatusOK)
		if err := json.Unmarshal([]byte(body), &pack); err != nil {
			t.Fatal(err)
		}
		if pack.RevisionVersion != version || pack.Content == nil || pack.Content.Mission.Title != fmt.Sprintf("Practice v%d", version) {
			t.Fatalf("wrong published practice: %+v", pack)
		}
	}

	// A published practice must not expose a passport that is still a draft.
	request(http.MethodGet, practice, "", portal.learner, http.StatusNotFound)
	request(http.MethodPost, passportVersion(1, "publish"), "", portal.lead, http.StatusOK)
	assertPractice(1) // v2's practice has a newer publication time but belongs to a draft.

	request(http.MethodPost, passportVersion(2, "learning-pack/unpublish"), "", portal.lead, http.StatusOK)
	request(http.MethodPost, passportVersion(2, "publish"), "", portal.lead, http.StatusOK)
	request(http.MethodGet, practice, "", portal.learner, http.StatusNotFound)
	request(http.MethodPost, passportVersion(2, "learning-pack/publish"), "", portal.lead, http.StatusOK)
	assertPractice(2)

	// Rolling back selects the old practice; unpublishing hides practice as well.
	request(http.MethodPost, passportVersion(1, "publish"), "", portal.lead, http.StatusOK)
	assertPractice(1)
	request(http.MethodPost, passportVersion(1, "unpublish"), "", portal.lead, http.StatusOK)
	request(http.MethodGet, practice, "", portal.learner, http.StatusNotFound)
}

func TestPracticePublicationChangesOnlyWithContent(t *testing.T) {
	t.Parallel()
	portal := newTestPortal(t)
	seedPracticeVersions(t, portal)
	publish := func() learningpack.View {
		t.Helper()
		body := portal.request(http.MethodPost, "/api/passport-revisions/revision-1/learning-pack/publish", "", portal.lead, http.StatusOK)
		var pack learningpack.View
		if err := json.Unmarshal([]byte(body), &pack); err != nil {
			t.Fatal(err)
		}
		return pack
	}
	// Repeating publication of unchanged content must keep the learner's progress key.
	first := publish()
	if first.PublishedAt != "2026-10-16T11:00:00Z" {
		t.Fatalf("unchanged practice got a new publication: %s", first.PublishedAt)
	}
	if _, err := portal.database.ExecContext(t.Context(), `UPDATE learning_packs SET content_json=json_set(content_json,'$.mission.title','Updated practice') WHERE revision_id='revision-1'`); err != nil {
		t.Fatal(err)
	}
	updated := publish()
	if updated.PublishedAt == first.PublishedAt || updated.Content == nil || updated.Content.Mission.Title != "Updated practice" {
		t.Fatalf("changed practice kept its old publication: %+v", updated)
	}
	if repeated := publish(); repeated.PublishedAt != updated.PublishedAt {
		t.Fatalf("repeated publication reset progress: %s != %s", repeated.PublishedAt, updated.PublishedAt)
	}
}
