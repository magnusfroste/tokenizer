package classifier_test

import (
	"testing"

	"github.com/magnusfroste/tokenizer/internal/classifier"
	"github.com/magnusfroste/tokenizer/internal/openai"
	"github.com/magnusfroste/tokenizer/internal/router"
)

func TestClassifyTaskGoldenCases(t *testing.T) {
	tests := []struct {
		name       string
		messages   []openai.Message
		wantTask   router.TaskType
		wantSignal string
	}{
		{
			name: "commit message diff is trivial git",
			messages: []openai.Message{{
				Role: "user",
				Content: "Write a commit message for this diff:\n" +
					"diff --git a/internal/router/job.go b/internal/router/job.go\n" +
					"@@ -1 +1 @@\n-old\n+new",
			}},
			wantTask:   router.TaskTrivialGit,
			wantSignal: "git_keyword",
		},
		{
			name:       "short general question is simple chat",
			messages:   []openai.Message{{Role: "user", Content: "What is the difference between latency and throughput?"}},
			wantTask:   router.TaskSimpleChat,
			wantSignal: "short_prompt",
		},
		{
			name:       "summarize request is summarization",
			messages:   []openai.Message{{Role: "user", Content: "Summarize this incident log into three bullets."}},
			wantTask:   router.TaskSummarization,
			wantSignal: "summarize_keyword",
		},
		{
			name: "small code edit is simple code edit",
			messages: []openai.Message{{
				Role:    "user",
				Content: "Update internal/router/job.go to rename `buildJob` to `newJob`.",
			}},
			wantTask:   router.TaskSimpleCodeEdit,
			wantSignal: "edit_keyword",
		},
		{
			name: "stack trace auth path is hard code debugging",
			messages: []openai.Message{{
				Role: "user",
				Content: "Fix this auth crash in src/auth/session.ts:\n" +
					"TypeError: cannot read properties of undefined\n" +
					"    at handler (/app/src/auth/session.ts:10:5)",
			}},
			wantTask:   router.TaskHardCodeDebugging,
			wantSignal: "stack_trace",
		},
		{
			name:       "prose race condition debugging is hard code debugging",
			messages:   []openai.Message{{Role: "user", Content: "Debug this hard race condition deadlock in my concurrent Go code, stack trace attached."}},
			wantTask:   router.TaskHardCodeDebugging,
			wantSignal: "debug_keyword",
		},
		{
			name:       "prose segfault debugging is hard code debugging",
			messages:   []openai.Message{{Role: "user", Content: "My service keeps crashing with a segfault and a nil pointer dereference; help me debug it."}},
			wantTask:   router.TaskHardCodeDebugging,
			wantSignal: "debug_keyword",
		},
		{
			name: "migration sql is database migration",
			messages: []openai.Message{{
				Role: "user",
				Content: "Create db/migrations/004_add_users.sql with:\n" +
					"```sql\nALTER TABLE users ADD COLUMN deleted_at timestamptz;\n```",
			}},
			wantTask:   router.TaskDatabaseMigration,
			wantSignal: "database_migration_keyword",
		},
		{
			name:       "xss secret security review is security review",
			messages:   []openai.Message{{Role: "user", Content: "Security review this login form for XSS and secret leakage."}},
			wantTask:   router.TaskSecurityReview,
			wantSignal: "security_review_keyword",
		},
		{
			name:       "unknown production secret pii hint is high risk unknown",
			messages:   []openai.Message{{Role: "user", Content: "The production customer PII includes an API key. What now?"}},
			wantTask:   router.TaskUnknownHighRisk,
			wantSignal: "low_task_confidence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := classifier.ExtractFromMessages(tt.messages, classifier.RequestHints{})
			got := classifier.ClassifyTask(features, tt.messages)

			if got.TaskType != string(tt.wantTask) {
				t.Fatalf("task type mismatch: got %q want %q; classification=%+v features=%+v", got.TaskType, tt.wantTask, got, features)
			}
			if got.Confidence < 0 || got.Confidence > 1 {
				t.Fatalf("confidence out of range: %+v", got)
			}
			if got.Confidence == 0 {
				t.Fatalf("expected non-zero confidence: %+v", got)
			}
			if len(got.Signals) == 0 {
				t.Fatalf("expected signals: %+v", got)
			}
			assertSignal(t, got.Signals, tt.wantSignal)
		})
	}
}

func TestClassifyTaskSpecificRiskWinsOverGenericCodeEdit(t *testing.T) {
	messages := []openai.Message{{
		Role: "user",
		Content: "Update db/migrations/007_accounts.sql:\n" +
			"```sql\nALTER TABLE accounts ADD COLUMN plan text;\n```",
	}}
	features := classifier.ExtractFromMessages(messages, classifier.RequestHints{})

	got := classifier.ClassifyTask(features, messages)

	if got.TaskType != string(router.TaskDatabaseMigration) {
		t.Fatalf("expected migration to win over generic edit, got %+v", got)
	}
	assertSignal(t, got.Signals, "database_migration_keyword")
}

func assertSignal(t *testing.T, signals []string, want string) {
	t.Helper()
	for _, signal := range signals {
		if signal == want {
			return
		}
	}
	t.Fatalf("expected signal %q in %#v", want, signals)
}
